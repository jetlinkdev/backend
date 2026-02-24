package intents

import (
	"context"
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/models"
	"jetlink/redis"
	"jetlink/utils"
)

// HandleAuth handles user authentication (login/register)
func HandleAuth(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	userRepo := database.NewUserRepository(repo.GetDB())

	// Extract auth data
	authData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for auth intent")
		sendError(client, "Invalid data format", incomingMsg)
		return
	}

	// Extract Firebase UID
	firebaseUID, ok := authData["uid"].(string)
	if !ok || firebaseUID == "" {
		logger.Error("Missing or invalid Firebase UID")
		sendError(client, "Missing Firebase UID", incomingMsg)
		return
	}

	// Check if user exists by Firebase UID
	existingUser, err := userRepo.GetUserByID(firebaseUID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		sendError(client, "Failed to authenticate user", incomingMsg)
		return
	}

	if existingUser != nil {
		// User exists, update last login
		err = userRepo.UpdateLastLogin(firebaseUID)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to update last login: %v", err))
		}
		logger.Info(fmt.Sprintf("User logged in: %s (%s)", existingUser.Email, existingUser.Role))

		// Associate client with user
		hub.AssociateClientWithUser(client, firebaseUID)

		// Check if user has active order and sync state
		userState := hub.GetUserOrderState(firebaseUID)
		if userState != nil {
			// Restore OrderID to client session (critical for cancel_order to work after reload)
			client.OrderID = &userState.OrderID
			logger.Info(fmt.Sprintf("Restored OrderID %d to client %s session", userState.OrderID, client.ID))

			// Update Redis client-order mappings for this new client connection
			if hub.OrderRedis != nil {
				ctx := context.Background()

				// Store client -> order mapping
				err := hub.OrderRedis.GetClient().InnerClient().Set(
					ctx,
					fmt.Sprintf("client:order:%s", client.ID),
					userState.OrderID,
					redis.OrderTTL,
				).Err()
				if err != nil {
					logger.Error(fmt.Sprintf("Failed to store client-order mapping in Redis: %v", err))
				}

				// Update order -> client mapping to new client ID
				err = hub.OrderRedis.GetClient().InnerClient().Set(
					ctx,
					fmt.Sprintf("order:client:%d", userState.OrderID),
					client.ID,
					redis.OrderTTL,
				).Err()
				if err != nil {
					logger.Error(fmt.Sprintf("Failed to update order-client mapping in Redis: %v", err))
				}
			}

			// Get full order data from database
			order, err := repo.GetOrder(userState.OrderID)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to get order %d for user %s: %v", userState.OrderID, firebaseUID, err))
			} else {
				// Get bids from Redis if available
				var bidsData []map[string]interface{}
				if hub.BidRedis != nil {
					ctx := context.Background()
					orderBids, err := hub.BidRedis.GetOrderBids(ctx, userState.OrderID)
					if err != nil {
						logger.Error(fmt.Sprintf("Failed to get bids for order %d: %v", userState.OrderID, err))
					} else {
						for _, bid := range orderBids {
							// Get driver info for each bid
							driver, err := userRepo.GetUserByID(bid.DriverID)
							if err != nil {
								logger.Error(fmt.Sprintf("Failed to get driver %s info for bid: %v", bid.DriverID, err))
							}

							bidData := map[string]interface{}{
								"bid_id":                 bid.ID,
								"order_id":               bid.OrderID,
								"driver_id":              bid.DriverID,
								"bid_price":              bid.BidPrice,
								"eta_minutes":            bid.ETAMinutes,
								"estimated_arrival_time": bid.EstimatedArrivalTime,
								"status":                 bid.Status,
							}

							// Include driver info if available
							if driver != nil {
								bidData["driver_name"] = driver.DisplayName
								bidData["rating"] = driver.DriverRating
								bidData["vehicle"] = driver.VehicleType
								bidData["plate_number"] = driver.VehiclePlate
							}

							bidsData = append(bidsData, bidData)
						}
					}
				}

				// Sync existing order state and data to this connection
				syncMsg := hubhandlers.Message{
					Intent: "order_state_sync",
					Data: map[string]interface{}{
						"order_id":                 order.ID,
						"status":                   order.Status,
						"ui_state":                 userState.UIState,
						"pickup":                   order.Pickup,
						"pickup_latitude":          order.PickupLatitude,
						"pickup_longitude":         order.PickupLongitude,
						"destination":              order.Destination,
						"destination_latitude":     order.DestinationLatitude,
						"destination_longitude":    order.DestinationLongitude,
						"notes":                    order.Notes,
						"payment":                  order.Payment,
						"fare":                     order.Fare,
						"bid_price":                order.BidPrice,
						"estimated_arrival_time":   order.EstimatedArrivalTime,
						"bids":                     bidsData,
					},
					Timestamp: time.Now().Unix(),
				}
				client.Send <- syncMsg.ToJSON()
				logger.Info(fmt.Sprintf("Synced order %d state to user %s: %s", order.ID, firebaseUID, userState.UIState))
			}
		}

		// Send success response with user data
		successMsg := hubhandlers.Message{
			Intent: constants.IntentAuthSuccess,
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":           existingUser.ID,
					"email":        existingUser.Email,
					"displayName":  existingUser.DisplayName,
					"photoURL":     existingUser.PhotoURL,
					"role":         existingUser.Role,
					"isVerified":   existingUser.IsVerified,
					"vehicleType":  existingUser.VehicleType,
					"vehiclePlate": existingUser.VehiclePlate,
				},
				"message": "User already exists",
			},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- successMsg.ToJSON()
		return
	}

	// User doesn't exist by Firebase UID, check if we have complete profile data
	email, _ := authData["email"].(string)
	displayName, _ := authData["displayName"].(string)
	photoURL, _ := authData["photoURL"].(string)
	phoneNumber, _ := authData["phoneNumber"].(string)

	// If we have all required data, check if user exists by email first
	if email != "" && displayName != "" {
		// Check if user already exists by email
		existingUserByEmail, err := userRepo.GetUserByEmail(email)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to check existing user by email: %v", err))
		}

		if existingUserByEmail != nil {
			// User already exists with this email, update Firebase UID
			logger.Info(fmt.Sprintf("User already exists with email %s, updating Firebase UID", email))
			
			existingUserByEmail.ID = firebaseUID
			existingUserByEmail.PhotoURL = photoURL
			if phoneNumber != "" {
				existingUserByEmail.PhoneNumber = phoneNumber
			}
			existingUserByEmail.UpdatedAt = time.Now().Unix()
			
			err = userRepo.UpdateUser(existingUserByEmail)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to update user Firebase UID: %v", err))
			}

			// Update last login
			userRepo.UpdateLastLogin(firebaseUID)

			// Send success response
			successMsg := hubhandlers.Message{
				Intent: constants.IntentAuthSuccess,
				Data: map[string]interface{}{
					"user": map[string]interface{}{
						"id":           existingUserByEmail.ID,
						"email":        existingUserByEmail.Email,
						"displayName":  existingUserByEmail.DisplayName,
						"photoURL":     existingUserByEmail.PhotoURL,
						"role":         existingUserByEmail.Role,
						"isVerified":   existingUserByEmail.IsVerified,
						"vehicleType":  existingUserByEmail.VehicleType,
						"vehiclePlate": existingUserByEmail.VehiclePlate,
					},
					"message": "User already exists",
				},
				Timestamp: time.Now().Unix(),
				ClientID:  client.ID,
			}
			client.Send <- successMsg.ToJSON()
			return
		}

		// User doesn't exist, create new user
		user := &models.User{
			ID:            firebaseUID,
			Email:         email,
			DisplayName:   displayName,
			PhotoURL:      photoURL,
			PhoneNumber:   phoneNumber,
			Role:          "customer",
			DriverRating:  0.0,
			TotalTrips:    0,
			IsVerified:    false,
			CreatedAt:     time.Now().Unix(),
			UpdatedAt:     time.Now().Unix(),
		}

		err = userRepo.CreateUser(user)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create user: %v", err))
			sendError(client, fmt.Sprintf("Failed to create user: %v", err), incomingMsg)
			return
		}

		logger.Info(fmt.Sprintf("New customer registered: %s", email))

		// Send success response
		successMsg := hubhandlers.Message{
			Intent: constants.IntentAuthSuccess,
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":           user.ID,
					"email":        user.Email,
					"displayName":  user.DisplayName,
					"photoURL":     user.PhotoURL,
					"role":         user.Role,
					"isVerified":   user.IsVerified,
					"vehicleType":  user.VehicleType,
					"vehiclePlate": user.VehiclePlate,
				},
				"message": "User created successfully",
			},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- successMsg.ToJSON()
		return
	}

	// Profile data incomplete, request complete profile
	logger.Info(fmt.Sprintf("User %s needs to complete profile", firebaseUID))

	profileMsg := hubhandlers.Message{
		Intent: constants.IntentAuthProfileNeeded,
		Data: map[string]interface{}{
			"uid":     firebaseUID,
			"message": "Please complete your profile",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- profileMsg.ToJSON()
}

// HandleCompleteProfile handles completing user profile after initial login
func HandleCompleteProfile(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	userRepo := database.NewUserRepository(repo.GetDB())

	// Extract profile data
	profileData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for complete_profile intent")
		sendError(client, "Invalid data format", incomingMsg)
		return
	}

	// Extract Firebase UID
	firebaseUID, ok := profileData["uid"].(string)
	if !ok || firebaseUID == "" {
		logger.Error("Missing or invalid Firebase UID")
		sendError(client, "Missing Firebase UID", incomingMsg)
		return
	}

	// Extract required profile fields
	email, ok := profileData["email"].(string)
	if !ok || email == "" {
		sendError(client, "Missing email", incomingMsg)
		return
	}

	displayName, ok := profileData["displayName"].(string)
	if !ok || displayName == "" {
		sendError(client, "Missing display name", incomingMsg)
		return
	}

	photoURL, _ := profileData["photoURL"].(string)
	phoneNumber, _ := profileData["phoneNumber"].(string)

	// Check if user already exists by email (email = unique identifier)
	existingUser, err := userRepo.GetUserByEmail(email)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to check existing user: %v", err))
	}

	if existingUser != nil {
		// User already exists with this email
		// Update Firebase UID to link this Firebase account
		existingUser.ID = firebaseUID
		existingUser.PhotoURL = photoURL
		if phoneNumber != "" {
			existingUser.PhoneNumber = phoneNumber
		}
		existingUser.UpdatedAt = time.Now().Unix()
		
		err = userRepo.UpdateUser(existingUser)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to update user Firebase UID: %v", err))
		}

		logger.Info(fmt.Sprintf("User already exists with email %s, updated Firebase UID", email))

		// Send success response - user already registered
		successMsg := hubhandlers.Message{
			Intent: constants.IntentAuthSuccess,
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":           existingUser.ID,
					"email":        existingUser.Email,
					"displayName":  existingUser.DisplayName,
					"photoURL":     existingUser.PhotoURL,
					"role":         existingUser.Role,
					"isVerified":   existingUser.IsVerified,
					"vehicleType":  existingUser.VehicleType,
					"vehiclePlate": existingUser.VehiclePlate,
				},
				"message": "User already registered",
			},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- successMsg.ToJSON()
		return
	}

	// User doesn't exist, create new user with complete profile
	user := &models.User{
		ID:            firebaseUID,
		Email:         email,
		DisplayName:   displayName,
		PhotoURL:      photoURL,
		PhoneNumber:   phoneNumber,
		Role:          "customer",
		DriverRating:  0.0,
		TotalTrips:    0,
		IsVerified:    false,
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}

	err = userRepo.CreateUser(user)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create user: %v", err))
		sendError(client, fmt.Sprintf("Failed to create user: %v", err), incomingMsg)
		return
	}

	logger.Info(fmt.Sprintf("New customer registered with complete profile: %s", email))

	// Send success response
	successMsg := hubhandlers.Message{
		Intent: constants.IntentAuthSuccess,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":           user.ID,
				"email":        user.Email,
				"displayName":  user.DisplayName,
				"photoURL":     user.PhotoURL,
				"role":         user.Role,
				"isVerified":   user.IsVerified,
				"vehicleType":  user.VehicleType,
				"vehiclePlate": user.VehiclePlate,
			},
			"message": "Profile completed successfully",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()
}

// HandleDriverRegistration handles driver registration
func HandleDriverRegistration(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	userRepo := database.NewUserRepository(repo.GetDB())

	// Extract registration data
	regData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for driver_registration intent")
		sendError(client, "Invalid data format", incomingMsg)
		return
	}

	// Extract Firebase UID
	firebaseUID, ok := regData["uid"].(string)
	if !ok || firebaseUID == "" {
		logger.Error("Missing or invalid Firebase UID")
		sendError(client, "Missing Firebase UID", incomingMsg)
		return
	}

	// Check if user exists
	user, err := userRepo.GetUserByID(firebaseUID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		sendError(client, "Failed to get user", incomingMsg)
		return
	}

	if user == nil {
		logger.Error("User not found")
		sendError(client, "User not found. Please login first.", incomingMsg)
		return
	}

	// Check if already registered as driver
	if user.Role == "driver" {
		logger.Error("User is already registered as driver")
		sendError(client, "Already registered as driver", incomingMsg)
		return
	}

	// Extract driver data
	vehicleType, ok := regData["vehicleType"].(string)
	if !ok || vehicleType == "" {
		sendError(client, "Missing vehicle type", incomingMsg)
		return
	}

	vehiclePlate, ok := regData["vehiclePlate"].(string)
	if !ok || vehiclePlate == "" {
		sendError(client, "Missing vehicle plate", incomingMsg)
		return
	}

	// Update user with driver info
	user.VehicleType = vehicleType
	user.VehiclePlate = vehiclePlate
	user.Role = "driver"
	user.IsVerified = true
	user.UpdatedAt = time.Now().Unix()

	err = userRepo.RegisterDriver(user)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to register driver: %v", err))
		sendError(client, "Failed to register driver", incomingMsg)
		return
	}

	logger.Info(fmt.Sprintf("Driver registered: %s (%s)", user.Email, vehiclePlate))

	// Send success response
	successMsg := hubhandlers.Message{
		Intent: constants.IntentDriverRegistered,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":           user.ID,
				"email":        user.Email,
				"displayName":  user.DisplayName,
				"role":         "driver",
				"isVerified":   true,
				"vehicleType":  user.VehicleType,
				"vehiclePlate": user.VehiclePlate,
			},
			"message": "Driver registration successful",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()
}

// HandleCheckDriverStatus checks if a user is registered as a driver
func HandleCheckDriverStatus(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	userRepo := database.NewUserRepository(repo.GetDB())

	// Extract Firebase UID
	checkData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for check_driver_status intent")
		sendError(client, "Invalid data format", incomingMsg)
		return
	}

	firebaseUID, ok := checkData["uid"].(string)
	if !ok || firebaseUID == "" {
		logger.Error("Missing or invalid Firebase UID")
		sendError(client, "Missing Firebase UID", incomingMsg)
		return
	}

	// Check if user exists and is a driver
	user, err := userRepo.GetUserByID(firebaseUID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		sendError(client, "Failed to check driver status", incomingMsg)
		return
	}

	isDriver := false
	isVerified := false
	var vehicleType, vehiclePlate string

	if user != nil {
		isDriver = user.Role == "driver"
		isVerified = user.IsVerified
		vehicleType = user.VehicleType
		vehiclePlate = user.VehiclePlate
	}

	// Send response
	successMsg := hubhandlers.Message{
		Intent: constants.IntentDriverStatus,
		Data: map[string]interface{}{
			"isDriver":     isDriver,
			"isVerified":   isVerified,
			"vehicleType":  vehicleType,
			"vehiclePlate": vehiclePlate,
			"message":      "Driver status checked",
		},
		Timestamp: time.Now().Unix(),
	}
	client.Send <- successMsg.ToJSON()
}

func sendError(client *hubhandlers.Client, message string, incomingMsg hubhandlers.Message) {
	errorMsg := hubhandlers.Message{
		Intent:    constants.IntentError,
		Data:      map[string]string{"message": message},
		Timestamp: time.Now().Unix(),
		ClientID:  incomingMsg.ClientID,
	}
	client.Send <- errorMsg.ToJSON()
}
