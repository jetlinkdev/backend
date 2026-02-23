package intents

import (
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/utils"
)

// HandleCompleteTrip handles the complete_trip intent
func HandleCompleteTrip(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Extract the trip completion data from the incoming message
	completeData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for complete_trip intent")

		// Send error response back to client
		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for complete_trip"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract order ID from the message
	orderIDFloat, ok := completeData["order_id"].(float64)
	if !ok {
		logger.Error("Missing or invalid order_id in complete_trip request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid order_id in complete_trip request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	orderID := int64(orderIDFloat)

	// Extract driver ID
	driverID, ok := completeData["driver_id"].(string)
	if !ok || driverID == "" {
		logger.Error("Missing or invalid driver_id in complete_trip request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid driver_id in complete_trip request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Retrieve the order from the database
	order, err := repo.GetOrder(orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve order %d for trip completion: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order not found"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if the order status allows completion (driver_arrived or in_progress)
	if order.Status != "driver_arrived" && order.Status != "in_progress" {
		logger.Error(fmt.Sprintf("Order %d is not in a state that allows trip completion (current status: %s)", orderID, order.Status))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Trip can only be completed after driver has arrived and trip has started"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if the driver is the one assigned to this order
	if order.DriverID != driverID {
		logger.Error(fmt.Sprintf("Driver %s is not assigned to order %d", driverID, orderID))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "You are not the assigned driver for this order"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Update the order status to completed
	order.Status = "completed"
	order.UpdatedAt = time.Now().Unix()
	if err := repo.UpdateOrder(order); err != nil {
		logger.Error(fmt.Sprintf("Failed to update order %d status to completed: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to update order status"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	logger.Info(fmt.Sprintf("Trip completed for order %d by driver %s. Fare: Rp %.0f", orderID, driverID, order.Fare))

	// Send success response back to the driver
	successMsg := hubhandlers.Message{
		Intent: constants.IntentTripCompleted,
		Data: map[string]interface{}{
			"order_id":  orderID,
			"driver_id": driverID,
			"status":    "completed",
			"fare":      order.Fare,
			"message":   "Trip completed successfully. Thank you for your service!",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()

	// Broadcast trip completion notification to all clients (including the passenger)
	broadcastMsg := hubhandlers.Message{
		Intent: constants.IntentTripCompleted,
		Data: map[string]interface{}{
			"order_id":     orderID,
			"driver_id":    driverID,
			"status":       "completed",
			"fare":         order.Fare,
			"pickup":       order.Pickup,
			"destination":  order.Destination,
			"payment":      order.Payment,
			"message":      "Your trip has been completed. Thank you for using Jetlink!",
		},
		Timestamp: time.Now().Unix(),
	}
	hub.BroadcastMessage(broadcastMsg)
}
