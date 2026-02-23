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

// HandleSubmitBid handles the submit_bid intent
func HandleSubmitBid(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Create bid repository
	bidRepo := database.NewBidRepository(repo.GetDB())

	// Extract the bid data from the incoming message
	bidData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for submit_bid intent")

		// Send error response back to client
		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for submit_bid"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract order ID from the message
	orderIDFloat, ok := bidData["order_id"].(float64)
	if !ok {
		logger.Error("Missing or invalid order_id in submit_bid request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid order_id in submit_bid request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	orderID := int64(orderIDFloat)

	// Extract driver ID
	driverID, ok := bidData["driver_id"].(string)
	if !ok || driverID == "" {
		logger.Error("Missing or invalid driver_id in submit_bid request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid driver_id in submit_bid request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract bid price
	bidPriceFloat, ok := bidData["bid_price"].(float64)
	if !ok || bidPriceFloat <= 0 {
		logger.Error("Missing or invalid bid_price in submit_bid request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid bid_price in submit_bid request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract estimated arrival time (in minutes from now)
	etaMinutesFloat, ok := bidData["estimated_arrival_time"].(float64)
	if !ok || etaMinutesFloat <= 0 {
		logger.Error("Missing or invalid estimated_arrival_time in submit_bid request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid estimated_arrival_time in submit_bid request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	
	// Calculate estimated arrival timestamp (current time + ETA minutes)
	etaTimestamp := time.Now().Unix() + int64(etaMinutesFloat*60)
	etaMinutes := int64(etaMinutesFloat)

	// Retrieve the order from the database
	order, err := repo.GetOrder(orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve order %d for bid: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order not found"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if order is still available for bidding
	if order.Status != "pending" {
		logger.Error(fmt.Sprintf("Order %d is not available for bidding (status: %s)", orderID, order.Status))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order is no longer available for bidding"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if driver has already placed a bid on this order
	hasBid, err := bidRepo.HasDriverBidForOrder(driverID, orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to check existing bid: %v", err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to check existing bids"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	if hasBid {
		logger.Error(fmt.Sprintf("Driver %s has already placed a bid on order %d", driverID, orderID))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "You have already placed a bid on this order"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Create a new bid
	bid := &models.Bid{
		OrderID:              orderID,
		DriverID:             driverID,
		BidPrice:             bidPriceFloat,
		EstimatedArrivalTime: etaTimestamp,
		ETAMinutes:           etaMinutes,
		Status:               "pending",
		CreatedAt:            time.Now().Unix(),
		UpdatedAt:            time.Now().Unix(),
	}

	// Store the bid in the database
	if err := bidRepo.CreateBid(bid); err != nil {
		logger.Error(fmt.Sprintf("Failed to store bid in database: %v", err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to submit bid"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	logger.Info(fmt.Sprintf("Driver %s submitted bid for order %d: Price=%.2f, ETA=%s", driverID, orderID, bidPriceFloat, time.Unix(etaTimestamp, 0).Format("2006-01-02 15:04:05")))

	// Send success response back to the driver
	successMsg := hubhandlers.Message{
		Intent: constants.IntentBidAccepted,
		Data: map[string]interface{}{
			"bid_id":                 bid.ID,
			"order_id":               orderID,
			"driver_id":              driverID,
			"bid_price":              bidPriceFloat,
			"estimated_arrival_time": etaTimestamp,
			"eta_minutes":            etaMinutes,
			"status":                 "pending",
			"message":                "Your bid has been submitted successfully",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()

	// Broadcast new bid notification to all clients (including the user who created the order)
	broadcastMsg := hubhandlers.Message{
		Intent: constants.IntentNewBidReceived,
		Data: map[string]interface{}{
			"bid_id":                 bid.ID,
			"order_id":               orderID,
			"driver_id":              driverID,
			"bid_price":              bidPriceFloat,
			"estimated_arrival_time": etaTimestamp,
			"eta_minutes":            etaMinutes,
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	hub.BroadcastMessage(broadcastMsg)
}
