package intents

import (
	"context"
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/utils"
)

// HandleGetMyBids handles the get_my_bids intent
func HandleGetMyBids(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Extract request data
	requestData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for get_my_bids intent")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for get_my_bids"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Get driver ID from client session
	driverID := client.UserID
	if driverID == "" {
		// Fallback: try to get from request data
		driverID, _ = requestData["driver_id"].(string)
	}

	if driverID == "" {
		logger.Error("No driver ID provided for get_my_bids")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Driver ID is required"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Get driver info
	userRepo := database.NewUserRepository(repo.GetDB())
	driver, err := userRepo.GetUserByID(driverID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get driver %s info: %v", driverID, err))
	}

	// Get bids from Redis
	var bidsData []map[string]interface{}
	if hub.BidRedis != nil {
		ctx := context.Background()

		// Get all bids for this driver by iterating through available orders
		// Note: This is a simplified approach. For production, consider storing driver->bids mapping
		availableOrderIDs, err := hub.OrderRedis.GetAvailableOrders(ctx)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get available orders: %v", err))
		} else {
			for _, orderID := range availableOrderIDs {
				orderBids, err := hub.BidRedis.GetOrderBids(ctx, orderID)
				if err != nil {
					continue
				}

				for _, bid := range orderBids {
					if bid.DriverID == driverID {
						// Get order info
						order, err := repo.GetOrder(orderID)
						if err != nil {
							continue
						}

						bidData := map[string]interface{}{
							"bid_id":                 bid.ID,
							"order_id":               bid.OrderID,
							"driver_id":              bid.DriverID,
							"bid_price":              bid.BidPrice,
							"eta_minutes":            bid.ETAMinutes,
							"estimated_arrival_time": bid.EstimatedArrivalTime,
							"status":                 bid.Status,
							"pickup":                 order.Pickup,
							"pickup_latitude":        order.PickupLatitude,
							"pickup_longitude":       order.PickupLongitude,
							"destination":            order.Destination,
							"destination_latitude":   order.DestinationLatitude,
							"destination_longitude":  order.DestinationLongitude,
							"fare":                   order.Fare,
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
		}
	}

	logger.Info(fmt.Sprintf("Driver %s requested my bids, found %d bids", driverID, len(bidsData)))

	// Send response
	successMsg := hubhandlers.Message{
		Intent: constants.IntentMyBids,
		Data: map[string]interface{}{
			"driver_id": driverID,
			"bids":      bidsData,
			"message":   fmt.Sprintf("Found %d bids", len(bidsData)),
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()
}
