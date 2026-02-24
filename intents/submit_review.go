package intents

import (
	"context"
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/models"
	"jetlink/utils"
)

// HandleSubmitReview handles the submit_review intent
func HandleSubmitReview(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Create review repository
	reviewRepo := database.NewReviewRepository(repo.GetDB())
	userRepo := database.NewUserRepository(repo.GetDB())

	// Extract the review data from the incoming message
	reviewData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for submit_review intent")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid data format for submit_review"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Extract order ID
	orderIDFloat, ok := reviewData["order_id"].(float64)
	if !ok {
		logger.Error("Missing or invalid order_id in submit_review request")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Missing or invalid order_id"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	orderID := int64(orderIDFloat)

	// Extract rating
	ratingFloat, ok := reviewData["rating"].(float64)
	if !ok || ratingFloat < 1 || ratingFloat > 5 {
		logger.Error("Missing or invalid rating in submit_review request (must be 1-5)")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Invalid rating. Must be between 1 and 5"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}
	rating := int(ratingFloat)

	// Extract optional review text
	reviewText, _ := reviewData["review"].(string)

	// Get user ID from client session
	userID := client.UserID
	if userID == "" {
		logger.Error("User not authenticated for submit_review")

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "User not authenticated"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Check if order already has a review
	hasReviewed, err := reviewRepo.HasReviewedOrder(orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to check existing review: %v", err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to check existing review"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	if hasReviewed {
		logger.Error(fmt.Sprintf("Order %d already has a review", orderID))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "You have already reviewed this order"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Get order to find driver ID
	order, err := repo.GetOrder(orderID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get order %d: %v", orderID, err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Order not found"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	if order.DriverID == "" {
		logger.Error(fmt.Sprintf("Order %d has no driver assigned", orderID))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "No driver assigned to this order"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	// Create the review
	review := &models.Review{
		OrderID:   orderID,
		UserID:    userID,
		DriverID:  order.DriverID,
		Rating:    rating,
		Review:    reviewText,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	if err := reviewRepo.CreateReview(review); err != nil {
		logger.Error(fmt.Sprintf("Failed to create review: %v", err))

		errorMsg := hubhandlers.Message{
			Intent:    constants.IntentError,
			Data:      map[string]string{"message": "Failed to submit review"},
			Timestamp: time.Now().Unix(),
			ClientID:  client.ID,
		}
		client.Send <- errorMsg.ToJSON()
		return
	}

	logger.Info(fmt.Sprintf("Review submitted for order %d: rating=%d, driver=%s", orderID, rating, order.DriverID))

	// Update driver's average rating
	avgRating, err := reviewRepo.GetDriverAverageRating(order.DriverID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get driver average rating: %v", err))
	}

	totalReviews, err := reviewRepo.GetDriverTotalReviews(order.DriverID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get driver total reviews: %v", err))
	}

	if err := userRepo.UpdateDriverRating(order.DriverID, avgRating, totalReviews); err != nil {
		logger.Error(fmt.Sprintf("Failed to update driver rating: %v", err))
	}

	// Send success response
	successMsg := hubhandlers.Message{
		Intent: constants.IntentReviewSubmitted,
		Data: map[string]interface{}{
			"review_id": review.ID,
			"order_id":  orderID,
			"rating":    rating,
			"message":   "Thank you for your review!",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()

	// Broadcast review notification to driver
	if hub.OrderRedis != nil {
		ctx := context.Background()
		driverClientID, _ := hub.OrderRedis.GetClientIDByOrderID(ctx, orderID)
		if driverClientID != "" {
			notificationMsg := hubhandlers.Message{
				Intent: "review_received",
				Data: map[string]interface{}{
					"order_id":  orderID,
					"rating":    rating,
					"review":    reviewText,
					"message":   "You received a new review!",
				},
				Timestamp: time.Now().Unix(),
			}
			hub.SendToClient(driverClientID, notificationMsg)
		}
	}
}
