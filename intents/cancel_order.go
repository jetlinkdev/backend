package intents

import (
	"context"
	"fmt"
	"time"

	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/utils"
	"jetlink/constants"
)

// HandleCancelOrder handles the cancel_order intent
func HandleCancelOrder(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Extract the order data from the incoming message
	orderData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for cancel_order intent")

		// Send error response back to client
		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for cancel_order"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Get order ID from client's session (stored in Hub/Redis)
	// Client doesn't need to send order_id anymore
	var orderID int64
	if client.OrderID != nil {
		orderID = *client.OrderID
	} else if hub.OrderRedis != nil {
		// Fallback: Try to get from Redis
		ctx := context.Background()
		order, err := hub.OrderRedis.GetOrderByClientID(ctx, client.ID)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get order from Redis for client %s: %v", client.ID, err))
		}
		if order != nil {
			orderID = order.ID
		}
	}

	if orderID == 0 {
		logger.Error("No active order found for client", client.ID)

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "No active order found to cancel"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract optional reason for cancellation (if provided)
	reason, _ := orderData["reason"].(string)

	// Retrieve the order from the database
	order, err := repo.GetOrder(orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve order %d for cancellation: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order not found or could not be retrieved"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if the order belongs to the current user or if it's cancellable
	// For now, we'll allow cancellation if the order exists and isn't already completed/cancelled
	if order.Status == "completed" || order.Status == "cancelled" {
		logger.Error(fmt.Sprintf("Order %d is already %s and cannot be cancelled", orderID, order.Status))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order is already completed or cancelled and cannot be cancelled again"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Update the order status to cancelled
	order.Status = "cancelled"
	order.UpdatedAt = time.Now().Unix()

	// Add cancellation reason if provided
	if reason != "" {
		// Note: Our current model doesn't have a reason field, but we could extend it if needed
		// For now, we just log the reason
		logger.Info(fmt.Sprintf("Order %d cancelled by client %s. Reason: %s", orderID, client.ID, reason))
	} else {
		logger.Info(fmt.Sprintf("Order %d cancelled by client %s", orderID, client.ID))
	}

	// Update the order in the database
	if err := repo.UpdateOrder(order); err != nil {
		logger.Error(fmt.Sprintf("Failed to update order %d status to cancelled: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to update order status in database"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Remove order from Redis
	if hub.OrderRedis != nil {
		err := hub.OrderRedis.DeleteActiveOrder(context.Background(), orderID, client.ID)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to delete order from Redis: %v", err))
		}
	}

	// Clear client-order mapping in Hub
	hub.ClearClientOrder(client)

	// Clear UserOrders state (user can create new order now)
	if client.UserID != "" {
		hub.ClearUserOrderState(client.UserID)
	}

	// Send success response back to the client who cancelled the order
	successMsg := hubhandlers.Message{
		Intent: constants.IntentOrderCancelled,
		Data: map[string]interface{}{
			"orderId": orderID,
			"status":  "cancelled",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()

	// Broadcast cancellation notification to other relevant clients (drivers, etc.)
	broadcastMsg := hubhandlers.Message{
		Intent: constants.IntentOrderCancelled,
		Data: map[string]interface{}{
			"orderId": orderID,
			"status":  "cancelled",
			"reason":  reason,
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	hub.BroadcastMessage(broadcastMsg)
}