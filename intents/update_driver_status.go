package intents

import (
	"fmt"
	"time"

	"jetlink/constants"
	"jetlink/database"
	hubhandlers "jetlink/handlers"
	"jetlink/utils"
)

// UpdateDriverStatusRequest represents the request body
type UpdateDriverStatusRequest struct {
	Status string `json:"status"` // "available" | "busy" | "offline"
}

// HandleUpdateDriverStatus handles the update_driver_status intent
func HandleUpdateDriverStatus(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, incomingMsg hubhandlers.Message, repo *database.OrderRepository) {
	// Extract status data
	statusData, ok := incomingMsg.Data.(map[string]interface{})
	if !ok {
		logger.Error("Invalid data format for update_driver_status intent")
		sendError(client, "Invalid data format", incomingMsg)
		return
	}

	// Get Firebase UID from client
	firebaseUID := client.UserID
	if firebaseUID == "" {
		logger.Error("User not authenticated")
		sendError(client, "User not authenticated", incomingMsg)
		return
	}

	// Extract new status
	newStatus, ok := statusData["status"].(string)
	if !ok || (newStatus != "available" && newStatus != "busy" && newStatus != "offline") {
		logger.Error("Invalid status value")
		sendError(client, "Invalid status. Must be 'available', 'busy', or 'offline'", incomingMsg)
		return
	}

	// Update driver status in database
	userRepo := database.NewUserRepository(repo.GetDB())
	err := userRepo.UpdateDriverStatus(firebaseUID, newStatus)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to update driver status: %v", err))
		sendError(client, "Failed to update status", incomingMsg)
		return
	}

	// Update client's DriverStatus field
	client.DriverStatus = newStatus

	logger.Info(fmt.Sprintf("Driver %s status updated to: %s", firebaseUID, newStatus))

	// Sync active orders if driver just went online
	if newStatus == "available" {
		logger.Info(fmt.Sprintf("Driver %s is now available, syncing active orders", firebaseUID))
		HandleSyncActiveOrders(client, hub, logger, repo)
	}

	// Send success response
	successMsg := hubhandlers.Message{
		Intent: constants.IntentDriverStatus,
		Data: map[string]interface{}{
			"status":  newStatus,
			"message": "Driver status updated successfully",
		},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- successMsg.ToJSON()
}
