package intents

import (
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/utils"
)

// HandleSyncActiveOrders sends active orders to a driver
func HandleSyncActiveOrders(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, repo *database.OrderRepository) {
	// Only drivers should receive active orders
	if client.Role != "driver" {
		logger.Warn(fmt.Sprintf("Non-driver client %s requested active orders", client.ID))
		return
	}

	// Get all active orders (pending status) from database
	orders, err := repo.GetOrdersByStatus("pending")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get active orders: %v", err))
		return
	}

	if len(orders) == 0 {
		logger.Info(fmt.Sprintf("No active orders to sync to driver %s", client.ID))
		return
	}

	logger.Info(fmt.Sprintf("Syncing %d active orders to driver %s", len(orders), client.ID))

	// Send each order to the driver
	for _, order := range orders {
		orderMsg := hubhandlers.Message{
			Intent:    constants.IntentNewOrderAvailable,
			Data:      order,
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- orderMsg.ToJSON()
	}

	logger.Info(fmt.Sprintf("Successfully synced %d active orders to driver %s", len(orders), client.ID))
}
