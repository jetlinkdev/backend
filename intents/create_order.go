package intents

import (
	"context"
	"fmt"
	"time"

	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/models"
	"jetlink/utils"
	"jetlink/constants"
)

// HandleCreateOrder handles the create_order intent
func HandleCreateOrder(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Extract the order data from the incoming message
	orderData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for create_order intent")

		// Send error response back to client
		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for create_order"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Get user ID from client session (set during auth)
	userID := client.UserID
	if userID == "" {
		logger.Error("User not authenticated (no UserID in client session)")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "User not authenticated"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if user already has active order
	existingState := hub.GetUserActiveOrder(userID)
	if existingState != nil {
		// User already has active order!
		logger.Info(fmt.Sprintf("User %s tried to create order while having active order %d",
			userID, existingState.OrderID))

		// Send existing order info (don't send error, send state sync)
		syncMsg := hubhandlers.Message{
			Intent: "existing_order_found",
			Data: map[string]interface{}{
				"order_id": existingState.OrderID,
				"status":   existingState.Status,
				"ui_state": existingState.UIState,
				"message":  "You already have an active order",
			},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- syncMsg.ToJSON()
		return
	}

	// Convert the order data to CreateOrderRequest struct
	createOrderReq := models.CreateOrderRequest{}

	// Extract values from the map
	if pickup, ok := orderData["pickup"].(string); ok {
		createOrderReq.Pickup = pickup
	} else {
		logger.Error("Missing or invalid pickup in create_order request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid pickup in create_order request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract pickup latitude and longitude
	pickupLat, ok := orderData["pickup_latitude"].(float64)
	if !ok {
		logger.Error("Missing or invalid pickup_latitude in create_order request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid pickup_latitude in create_order request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	createOrderReq.PickupLatitude = pickupLat

	pickupLng, ok := orderData["pickup_longitude"].(float64)
	if !ok {
		logger.Error("Missing or invalid pickup_longitude in create_order request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid pickup_longitude in create_order request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	createOrderReq.PickupLongitude = pickupLng

	if destination, ok := orderData["destination"].(string); ok {
		createOrderReq.Destination = destination
	} else {
		logger.Error("Missing or invalid destination in create_order request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid destination in create_order request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract destination latitude and longitude
	destLat, ok := orderData["destination_latitude"].(float64)
	if !ok {
		logger.Error("Missing or invalid destination_latitude in create_order request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid destination_latitude in create_order request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	createOrderReq.DestinationLatitude = destLat

	destLng, ok := orderData["destination_longitude"].(float64)
	if !ok {
		logger.Error("Missing or invalid destination_longitude in create_order request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid destination_longitude in create_order request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	createOrderReq.DestinationLongitude = destLng

	if notes, ok := orderData["notes"].(string); ok {
		createOrderReq.Notes = notes
	} else {
		// Notes are optional, so set to empty string if not provided
		createOrderReq.Notes = ""
	}

	// Handle optional time field - it can be null or a timestamp
	var timePtr *int64
	if timeVal, exists := orderData["time"]; exists && timeVal != nil {
		if timeFloat, ok := timeVal.(float64); ok {
			timeInt := int64(timeFloat)
			timePtr = &timeInt
		} else if timeStr, ok := timeVal.(string); ok {
			// If it's a string, try to parse it as a timestamp
			if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
				timeInt := parsedTime.Unix()
				timePtr = &timeInt
			} else {
				logger.Error("Invalid time format in create_order request")

				errorMsg := hubhandlers.Message{
					Intent:    constants.IntentError,
					Data:      map[string]string{"message": "Invalid time format in create_order request"},
					Timestamp: time.Now().Unix(),
					ClientID:  client.ID,
				}
				client.Send <- errorMsg.ToJSON()
				return
			}
		} else {
			logger.Error("Invalid time type in create_order request")

			errorMsg := hubhandlers.Message{
				Intent:    constants.IntentError,
				Data:      map[string]string{"message": "Invalid time type in create_order request"},
				Timestamp: time.Now().Unix(),
				ClientID:  client.ID,
			}
			client.Send <- errorMsg.ToJSON()
			return
		}
	} else {
		// Time is optional and can be nil
		timePtr = nil
	}
	createOrderReq.Time = timePtr

	// Extract payment
	payment, ok := orderData["payment"].(string)
	if !ok {
		// Payment is optional, so set to empty string if not provided
		payment = ""
	}
	createOrderReq.Payment = payment

	// Note: We don't use user_id from request data anymore
	// User ID is extracted from client session (set during auth)
	// The userID variable is already declared at the top of the function

	// Create a new order
	order := models.Order{
		UserID:               userID, // Use userID from client session
		Pickup:               createOrderReq.Pickup,
		PickupLatitude:       createOrderReq.PickupLatitude,
		PickupLongitude:      createOrderReq.PickupLongitude,
		Destination:          createOrderReq.Destination,
		DestinationLatitude:  createOrderReq.DestinationLatitude,
		DestinationLongitude: createOrderReq.DestinationLongitude,
		Notes:                createOrderReq.Notes,
		Time:                 createOrderReq.Time,
		Payment:              createOrderReq.Payment,
		Status:               "pending", // Initially pending
		CreatedAt:            time.Now().Unix(),
		UpdatedAt:            time.Now().Unix(),
	}

	// For now, set a default fare - in a real app this would be calculated based on distance/time
	order.Fare = 15000 // Default fare

	// Store the order in the database
	if err := repo.CreateOrder(&order); err != nil {
		logger.Error(fmt.Sprintf("Failed to store order in database: %v", err))

		// Send error response back to client
		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to create order in database"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	logger.Info(fmt.Sprintf("Created new order: %d for user: %s from %s to %s", order.ID, order.UserID, order.Pickup, order.Destination))

	// Store order in Redis for fast access
	if hub.OrderRedis != nil {
		err := hub.OrderRedis.CreateActiveOrder(context.Background(), &order, client.ID)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to store order in Redis: %v", err))
		}
	}

	// Set client-order mapping in Hub (also stores in Redis)
	hub.SetClientOrder(client, order.ID)

	// Update UserOrders state (for multi-tab sync)
	hub.SetUserOrderState(userID, order.ID, order.Status, "waiting_bids")

	// Send success response to ALL user's connections (including this one)
	successMsg := hubhandlers.Message{
		Intent:    constants.IntentOrderCreated,
		Data:      order.ID,
		Timestamp: time.Now().Unix(),
	}
	hub.BroadcastToUser(userID, successMsg)

	// Broadcast order notification to drivers (NOT to user's other tabs)
	broadcastMsg := hubhandlers.Message{
		Intent:    constants.IntentNewOrderAvailable,
		Data:      order,
		Timestamp: time.Now().Unix(),
	}

	// Send to all clients EXCEPT this user's connections
	hub.Mu.RLock()
	for c := range hub.Clients {
		if c.UserID != userID { // Skip user's own connections
			c.Send <- broadcastMsg.ToJSON()
		}
	}
	hub.Mu.RUnlock()
}