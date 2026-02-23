package intents

import (
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/models"
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

	// Extract email
	email, ok := authData["email"].(string)
	if !ok || email == "" {
		logger.Error("Missing or invalid email")
		sendError(client, "Missing email", incomingMsg)
		return
	}

	// Extract optional fields
	displayName, _ := authData["displayName"].(string)
	photoURL, _ := authData["photoURL"].(string)
	phoneNumber, _ := authData["phoneNumber"].(string)

	// Check if user exists
	existingUser, err := userRepo.GetUserByID(firebaseUID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		sendError(client, "Failed to authenticate user", incomingMsg)
		return
	}

	var user *models.User

	if existingUser != nil {
		// User exists, update last login
		user = existingUser
		err = userRepo.UpdateLastLogin(firebaseUID)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to update last login: %v", err))
		}
		logger.Info(fmt.Sprintf("User logged in: %s (%s)", user.Email, user.Role))
	} else {
		// Check if email already exists (user might have logged in before with different Firebase UID)
		existingEmailUser, err := userRepo.GetUserByEmail(email)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to check existing email: %v", err))
		}
		
		if existingEmailUser != nil {
			// User exists with different UID, update UID
			logger.Info(fmt.Sprintf("User with email %s found, updating UID", email))
			user = existingEmailUser
			err = userRepo.UpdateLastLogin(existingEmailUser.ID)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to update last login: %v", err))
			}
		} else {
			// New user, create customer by default
			user = &models.User{
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
				// Send more specific error message
				errorMsg := "Failed to create user"
				if err.Error() != "" {
					errorMsg = err.Error()
				}
				sendError(client, errorMsg, incomingMsg)
				return
			}

			logger.Info(fmt.Sprintf("New customer registered: %s", email))
		}
	}

	// Send success response
	successMsg := hubhandlers.Message{
		Intent: constants.IntentAuth,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":            user.ID,
				"email":         user.Email,
				"displayName":   user.DisplayName,
				"photoURL":      user.PhotoURL,
				"role":          user.Role,
				"isVerified":    user.IsVerified,
				"vehicleType":   user.VehicleType,
				"vehiclePlate":  user.VehiclePlate,
			},
			"message": "Authentication successful",
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
