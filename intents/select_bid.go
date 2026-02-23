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

// HandleSelectBid handles the select_bid intent
func HandleSelectBid(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Create bid repository
	bidRepo := database.NewBidRepository(repo.GetDB())

	// Extract the bid selection data from the incoming message
	selectData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for select_bid intent")

		// Send error response back to client
		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for select_bid"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract bid ID from the message
	bidIDFloat, ok := selectData["bid_id"].(float64)
	if !ok {
		logger.Error("Missing or invalid bid_id in select_bid request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid bid_id in select_bid request"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	bidID := int64(bidIDFloat)

	// Retrieve the bid from the database
	bid, err := bidRepo.GetBid(bidID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve bid %d for selection: %v", bidID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Bid not found"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Retrieve the order from the database
	order, err := repo.GetOrder(bid.OrderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve order %d for bid selection: %v", bid.OrderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order not found"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if order is still available for bid selection
	if order.Status != "pending" {
		logger.Error(fmt.Sprintf("Order %d is not available for bid selection (status: %s)", bid.OrderID, order.Status))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order is no longer available for bid selection"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if the bid is still pending
	if bid.Status != "pending" {
		logger.Error(fmt.Sprintf("Bid %d is no longer pending (status: %s)", bidID, bid.Status))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "This bid is no longer available for selection"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Update the selected bid status to accepted
	bid.Status = "accepted"
	bid.UpdatedAt = time.Now().Unix()
	if err := bidRepo.UpdateBid(bid); err != nil {
		logger.Error(fmt.Sprintf("Failed to update bid %d status to accepted: %v", bidID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to update bid status"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Reject all other bids for this order
	allBids, err := bidRepo.GetBidsByOrderID(bid.OrderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve bids for order %d: %v", bid.OrderID, err))
	} else {
		for _, otherBid := range allBids {
			if otherBid.ID != bidID && otherBid.Status == "pending" {
				otherBid.Status = "rejected"
				otherBid.UpdatedAt = time.Now().Unix()
				if err := bidRepo.UpdateBid(otherBid); err != nil {
					logger.Error(fmt.Sprintf("Failed to reject bid %d: %v", otherBid.ID, err))
				} else {
					// Notify other drivers that their bid was rejected
					rejectMsg := hubhandlers.Message{
						Intent: constants.IntentBidRejected,
						Data: map[string]interface{}{
							"bid_id":    otherBid.ID,
							"order_id":  otherBid.OrderID,
							"driver_id": otherBid.DriverID,
							"status":    "rejected",
							"message":   "Another driver was selected for this order",
						},
						Timestamp: time.Now().Unix(),
					}
					// Broadcast to all clients so the specific driver can receive it
					hub.BroadcastMessage(rejectMsg)
				}
			}
		}
	}

	// Update the order status to accepted and assign the driver
	order.Status = "accepted"
	order.DriverID = bid.DriverID
	order.BidPrice = bid.BidPrice
	order.EstimatedArrivalTime = &bid.EstimatedArrivalTime
	order.UpdatedAt = time.Now().Unix()
	if err := repo.UpdateOrder(order); err != nil {
		logger.Error(fmt.Sprintf("Failed to update order %d status to accepted: %v", bid.OrderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to update order status"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Update order status in Redis
	if hub.OrderRedis != nil {
		err := hub.OrderRedis.UpdateOrderStatus(context.Background(), order.ID, "accepted")
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to update order status in Redis: %v", err))
		}
	}

	logger.Info(fmt.Sprintf("Bid %d selected for order %d: Driver %s, Price=%.2f, ETA=%d minutes",
		bidID, bid.OrderID, bid.DriverID, bid.BidPrice, bid.ETAMinutes))

	// Send success response back to the user who selected the bid
	successMsg := hubhandlers.Message{
		Intent: constants.IntentSelectBid,
		Data: map[string]interface{}{
			"bid_id":                 bid.ID,
			"order_id":               bid.OrderID,
			"driver_id":              bid.DriverID,
			"bid_price":              bid.BidPrice,
			"estimated_arrival_time": bid.EstimatedArrivalTime,
			"eta_minutes":            bid.ETAMinutes,
			"status":                 "accepted",
			"message":                "Driver selected successfully",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()

	// Broadcast bid accepted notification to all clients (including the selected driver)
	broadcastMsg := hubhandlers.Message{
		Intent: constants.IntentBidAccepted,
		Data: map[string]interface{}{
			"bid_id":                 bid.ID,
			"order_id":               bid.OrderID,
			"driver_id":              bid.DriverID,
			"bid_price":              bid.BidPrice,
			"estimated_arrival_time": bid.EstimatedArrivalTime,
			"eta_minutes":            bid.ETAMinutes,
			"status":                 "accepted",
			"message":                "Your bid has been accepted! Please proceed to pickup location.",
		},
		Timestamp: time.Now().Unix(),
	}
	hub.BroadcastMessage(broadcastMsg)
}
