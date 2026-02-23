package intents

import (
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/utils"
)

// HandleDriverArrived handles the driver_arrived intent
func HandleDriverArrived(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Extract the arrival data from the incoming message
	arrivalData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for driver_arrived intent")

		// Send error response back to client
		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for driver_arrived"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract order ID from the message
	orderIDFloat, ok := arrivalData["order_id"].(float64)
	if !ok {
		logger.Error("Missing or invalid order_id in driver_arrived request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid order_id in driver_arrived request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	orderID := int64(orderIDFloat)

	// Extract driver ID
	driverID, ok := arrivalData["driver_id"].(string)
	if !ok || driverID == "" {
		logger.Error("Missing or invalid driver_id in driver_arrived request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid driver_id in driver_arrived request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Retrieve the order from the database
	order, err := repo.GetOrder(orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve order %d for driver arrival: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order not found"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if the order status is 'accepted' (driver should have been assigned)
	if order.Status != "accepted" && order.Status != "driver_arrived" {
		logger.Error(fmt.Sprintf("Order %d is not in accepted status (current status: %s)", orderID, order.Status))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Driver arrival can only be reported for accepted orders"},
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

	// Update the order status to driver_arrived
	order.Status = "driver_arrived"
	order.UpdatedAt = time.Now().Unix()
	if err := repo.UpdateOrder(order); err != nil {
		logger.Error(fmt.Sprintf("Failed to update order %d status to driver_arrived: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to update order status"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	logger.Info(fmt.Sprintf("Driver %s arrived at pickup location for order %d", driverID, orderID))

	// Send success response back to the driver
	successMsg := hubhandlers.Message{
		Intent: constants.IntentDriverArrived,
		Data: map[string]interface{}{
			"order_id":  orderID,
			"driver_id": driverID,
			"status":    "driver_arrived",
			"message":   "You have arrived at the pickup location. Please wait for the passenger.",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()

	// Broadcast driver arrival notification to all clients (including the passenger)
	broadcastMsg := hubhandlers.Message{
		Intent: constants.IntentDriverArrived,
		Data: map[string]interface{}{
			"order_id":  orderID,
			"driver_id": driverID,
			"status":    "driver_arrived",
			"message":   "Your driver has arrived at the pickup location!",
			"pickup":    order.Pickup,
		},
		Timestamp: time.Now().Unix(),
	}
	hub.BroadcastMessage(broadcastMsg)
}
