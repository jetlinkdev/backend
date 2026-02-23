package models

// Bid represents a driver's bid on an order
type Bid struct {
	ID                   int64   `json:"id"`
	OrderID              int64   `json:"orderId"`
	DriverID             string  `json:"driverId"`
	BidPrice             float64 `json:"bidPrice"`
	EstimatedArrivalTime int64   `json:"estimatedArrivalTime"` // Timestamp
	ETAMinutes           int64   `json:"etaMinutes"`           // Duration in minutes
	Status               string  `json:"status"`               // pending, accepted, rejected
	Message              string  `json:"message,omitempty"`
	CreatedAt            int64   `json:"createdAt"`
	UpdatedAt            int64   `json:"updatedAt"`
}

// CreateBidRequest represents the data for creating a new bid
type CreateBidRequest struct {
	OrderID              int64   `json:"orderId"`
	DriverID             string  `json:"driverId"`
	BidPrice             float64 `json:"bidPrice"`
	EstimatedArrivalTime int64   `json:"estimatedArrivalTime"` // Timestamp
	ETAMinutes           int64   `json:"etaMinutes"`           // Duration in minutes
}

// UpdateBidStatusRequest represents the data for updating a bid status
type UpdateBidStatusRequest struct {
	BidID   int64  `json:"bidId"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
