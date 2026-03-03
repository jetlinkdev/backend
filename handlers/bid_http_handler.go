package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"jetlink/database"
	"jetlink/models"
	"jetlink/utils"
)

// BidHTTPHandler handles HTTP REST API requests for bid operations
type BidHTTPHandler struct {
	logger     *utils.Logger
	bidRepo    *database.BidRepository
	orderRepo  *database.OrderRepository
	userRepo   *database.UserRepository
	hub        *Hub
}

// NewBidHTTPHandler creates a new BidHTTPHandler
func NewBidHTTPHandler(logger *utils.Logger, db *database.DB, hub *Hub) *BidHTTPHandler {
	return &BidHTTPHandler{
		logger:    logger,
		bidRepo:   database.NewBidRepository(db),
		orderRepo: database.NewOrderRepository(db),
		userRepo:  database.NewUserRepository(db),
		hub:       hub,
	}
}

// SubmitBidRequest represents the request body for submitting a bid
type SubmitBidRequest struct {
	OrderID    int64   `json:"orderId"`
	BidPrice   float64 `json:"bidPrice"`
	ETAMinutes int64   `json:"etaMinutes"`
}

// SubmitBidResponse represents the response for bid submission
type SubmitBidResponse struct {
	BidID        int64   `json:"bidId"`
	OrderID      int64   `json:"orderId"`
	DriverID     string  `json:"driverId"`
	BidPrice     float64 `json:"bidPrice"`
	ETAMinutes   int64   `json:"etaMinutes"`
	Status       string  `json:"status"`
	CreatedAt    int64   `json:"createdAt"`
}

// SubmitBid handles POST /api/bids/submit
// Submits a bid for an order
func (h *BidHTTPHandler) SubmitBid(w http.ResponseWriter, r *http.Request) {
	// Only accept POST
	if r.Method != http.MethodPost {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get Firebase UID from context (set by middleware)
	firebaseUID, ok := r.Context().Value("firebaseUID").(string)
	if !ok || firebaseUID == "" {
		h.sendError(w, "Unauthorized - Missing Firebase UID", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req SubmitBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error(fmt.Sprintf("Failed to parse request: %v", err))
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.OrderID <= 0 {
		h.sendError(w, "Invalid order ID", http.StatusBadRequest)
		return
	}
	if req.BidPrice <= 0 {
		h.sendError(w, "Bid price must be greater than 0", http.StatusBadRequest)
		return
	}
	if req.ETAMinutes <= 0 {
		h.sendError(w, "ETA minutes must be greater than 0", http.StatusBadRequest)
		return
	}

	h.logger.Info(fmt.Sprintf("Driver %s submitting bid for order #%d: price=%.2f, eta=%d min",
		firebaseUID, req.OrderID, req.BidPrice, req.ETAMinutes))

	// Check if driver exists and is verified
	driver, err := h.userRepo.GetUserByID(firebaseUID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get driver: %v", err))
		h.sendError(w, "Failed to submit bid", http.StatusInternalServerError)
		return
	}
	if driver == nil || driver.Role != "driver" {
		h.sendError(w, "Only verified drivers can submit bids", http.StatusForbidden)
		return
	}
	if !driver.IsVerified {
		h.sendError(w, "Driver account is not verified", http.StatusForbidden)
		return
	}

	// Check if order exists
	order, err := h.orderRepo.GetOrder(req.OrderID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get order: %v", err))
		h.sendError(w, "Order not found", http.StatusNotFound)
		return
	}

	// Check if order is still pending
	if order.Status != "pending" {
		h.sendError(w, fmt.Sprintf("Cannot bid on order with status: %s", order.Status), http.StatusBadRequest)
		return
	}

	// Check if driver already placed a bid on this order
	hasBid, err := h.bidRepo.HasDriverBidForOrder(firebaseUID, req.OrderID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to check existing bid: %v", err))
		h.sendError(w, "Failed to submit bid", http.StatusInternalServerError)
		return
	}
	if hasBid {
		h.sendError(w, "You have already placed a bid on this order", http.StatusBadRequest)
		return
	}

	// Calculate estimated arrival timestamp
	estimatedArrivalTimestamp := time.Now().Unix() + (req.ETAMinutes * 60)

	// Create bid
	bid := &models.Bid{
		OrderID:              req.OrderID,
		DriverID:             firebaseUID,
		BidPrice:             req.BidPrice,
		EstimatedArrivalTime: estimatedArrivalTimestamp,
		ETAMinutes:           req.ETAMinutes,
		Status:               "pending",
		Message:              "",
		CreatedAt:            time.Now().Unix(),
		UpdatedAt:            time.Now().Unix(),
	}

	if err := h.bidRepo.CreateBid(bid); err != nil {
		h.logger.Error(fmt.Sprintf("Failed to create bid: %v", err))
		h.sendError(w, "Failed to submit bid", http.StatusInternalServerError)
		return
	}

	h.logger.Info(fmt.Sprintf("Bid submitted successfully: bidId=%d, orderId=%d, driverId=%s",
		bid.ID, bid.OrderID, bid.DriverID))

	// Broadcast new bid notification to order owner (customer) via WebSocket
	if h.hub != nil && order.UserID != "" {
		broadcastData := map[string]interface{}{
			"bid_id":                 bid.ID,
			"order_id":               bid.OrderID,
			"driver_id":              bid.DriverID,
			"bid_price":              bid.BidPrice,
			"estimated_arrival_time": bid.EstimatedArrivalTime,
			"eta_minutes":            bid.ETAMinutes,
		}

		// Include driver info if available
		if driver != nil {
			broadcastData["driver_name"] = driver.DisplayName
			broadcastData["rating"] = driver.DriverRating
			broadcastData["vehicle"] = driver.VehicleType
			broadcastData["plate_number"] = driver.VehiclePlate
		}

		broadcastMsg := Message{
			Intent:    "new_bid_received",
			Data:      broadcastData,
			Timestamp: time.Now().Unix(),
		}

		// Send to order owner (customer)
		h.hub.BroadcastToUser(order.UserID, broadcastMsg)
		h.logger.Info(fmt.Sprintf("Bid broadcast to order owner %s for order %d", order.UserID, order.ID))
	}

	// Return success response
	response := SubmitBidResponse{
		BidID:      bid.ID,
		OrderID:    bid.OrderID,
		DriverID:   bid.DriverID,
		BidPrice:   bid.BidPrice,
		ETAMinutes: bid.ETAMinutes,
		Status:     bid.Status,
		CreatedAt:  bid.CreatedAt,
	}

	h.sendSuccess(w, response, http.StatusCreated)
}

// GetOrderBids handles GET /api/bids/order/:orderId
// Gets all bids for a specific order
func (h *BidHTTPHandler) GetOrderBids(w http.ResponseWriter, r *http.Request) {
	// Only accept GET
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get Firebase UID from context (set by middleware)
	firebaseUID, ok := r.Context().Value("firebaseUID").(string)
	if !ok || firebaseUID == "" {
		h.sendError(w, "Unauthorized - Missing Firebase UID", http.StatusUnauthorized)
		return
	}

	// Get order ID from URL path
	vars := mux.Vars(r)
	orderIDStr := vars["orderId"]
	if orderIDStr == "" {
		h.sendError(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	// Parse order ID (implementation depends on mux.Vars)
	// For now, we'll use a simple approach
	var orderID int64
	fmt.Sscanf(orderIDStr, "%d", &orderID)

	if orderID <= 0 {
		h.sendError(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Check if the user is the owner of the order
	order, err := h.orderRepo.GetOrder(orderID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get order: %v", err))
		h.sendError(w, "Order not found", http.StatusNotFound)
		return
	}

	// Only the order owner can view bids
	if order.UserID != firebaseUID {
		h.sendError(w, "Unauthorized - You can only view bids for your own orders", http.StatusForbidden)
		return
	}

	// Get all bids for the order
	bids, err := h.bidRepo.GetBidsByOrderID(orderID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get bids: %v", err))
		h.sendError(w, "Failed to get bids", http.StatusInternalServerError)
		return
	}

	// Convert bids to response format
	type BidResponse struct {
		BidID        int64   `json:"bidId"`
		DriverID     string  `json:"driverId"`
		BidPrice     float64 `json:"bidPrice"`
		ETAMinutes   int64   `json:"etaMinutes"`
		Status       string  `json:"status"`
		DriverRating float64 `json:"driverRating,omitempty"`
		CreatedAt    int64   `json:"createdAt"`
	}

	var responses []BidResponse
	for _, bid := range bids {
		// Get driver info
		driver, err := h.userRepo.GetUserByID(bid.DriverID)
		driverRating := 0.0
		if err == nil && driver != nil {
			driverRating = driver.DriverRating
		}

		responses = append(responses, BidResponse{
			BidID:        bid.ID,
			DriverID:     bid.DriverID,
			BidPrice:     bid.BidPrice,
			ETAMinutes:   bid.ETAMinutes,
			Status:       bid.Status,
			DriverRating: driverRating,
			CreatedAt:    bid.CreatedAt,
		})
	}

	h.sendSuccess(w, map[string]interface{}{
		"orderId": orderID,
		"bids":    responses,
	}, http.StatusOK)
}

// GetMyBids handles GET /api/bids/my
// Gets all bids placed by the current driver
func (h *BidHTTPHandler) GetMyBids(w http.ResponseWriter, r *http.Request) {
	// Only accept GET
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get Firebase UID from context (set by middleware)
	firebaseUID, ok := r.Context().Value("firebaseUID").(string)
	if !ok || firebaseUID == "" {
		h.sendError(w, "Unauthorized - Missing Firebase UID", http.StatusUnauthorized)
		return
	}

	// Get all bids for the driver
	bids, err := h.bidRepo.GetBidsByDriverID(firebaseUID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get bids: %v", err))
		h.sendError(w, "Failed to get bids", http.StatusInternalServerError)
		return
	}

	// Convert bids to response format with order details
	type BidWithOrderResponse struct {
		BidID        int64   `json:"bidId"`
		OrderID      int64   `json:"orderId"`
		BidPrice     float64 `json:"bidPrice"`
		ETAMinutes   int64   `json:"etaMinutes"`
		Status       string  `json:"status"`
		Message      string  `json:"message,omitempty"`
		Pickup       string  `json:"pickup,omitempty"`
		Destination  string  `json:"destination,omitempty"`
		OrderStatus  string  `json:"orderStatus,omitempty"`
		CreatedAt    int64   `json:"createdAt"`
	}

	var responses []BidWithOrderResponse
	for _, bid := range bids {
		// Get order details
		order, err := h.orderRepo.GetOrder(bid.OrderID)
		pickup := ""
		destination := ""
		orderStatus := ""
		if err == nil && order != nil {
			pickup = order.Pickup
			destination = order.Destination
			orderStatus = order.Status
		}

		responses = append(responses, BidWithOrderResponse{
			BidID:       bid.ID,
			OrderID:     bid.OrderID,
			BidPrice:    bid.BidPrice,
			ETAMinutes:  bid.ETAMinutes,
			Status:      bid.Status,
			Message:     bid.Message,
			Pickup:      pickup,
			Destination: destination,
			OrderStatus: orderStatus,
			CreatedAt:   bid.CreatedAt,
		})
	}

	h.sendSuccess(w, map[string]interface{}{
		"driverId": firebaseUID,
		"bids":     responses,
	}, http.StatusOK)
}

// sendSuccess sends a successful JSON response
func (h *BidHTTPHandler) sendSuccess(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// sendError sends an error JSON response
func (h *BidHTTPHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
