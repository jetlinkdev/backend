package models

// Order represents a ride-hailing order
type Order struct {
	ID                    int64   `json:"id"`
	UserID                string  `json:"userId"`
	DriverID              string  `json:"driverId,omitempty"`
	Pickup                string  `json:"pickup"`
	PickupLatitude        float64 `json:"pickupLatitude"`
	PickupLongitude       float64 `json:"pickupLongitude"`
	Destination           string  `json:"destination"`
	DestinationLatitude   float64 `json:"destinationLatitude"`
	DestinationLongitude  float64 `json:"destinationLongitude"`
	Notes                 string  `json:"notes"`
	Time                  *int64  `json:"time,omitempty"` // Nullable timestamp
	Payment               string  `json:"payment"`
	Status                string  `json:"status"` // pending, accepted, in_progress, completed, cancelled
	Fare                  float64 `json:"fare,omitempty"`
	BidPrice              float64 `json:"bidPrice,omitempty"`
	EstimatedArrivalTime  *int64  `json:"estimatedArrivalTime,omitempty"` // Timestamp waktu tiba di tempat jemput
	RouteCoordinates      string  `json:"routeCoordinates,omitempty"`     // Route coordinates from OSRM (stringified array)
	CreatedAt             int64   `json:"createdAt"`
	UpdatedAt             int64   `json:"updatedAt"`
	DeletedAt             *int64  `json:"deletedAt,omitempty"`
}

// CreateOrderRequest represents the data for creating a new order
type CreateOrderRequest struct {
	Pickup               string  `json:"pickup"`
	PickupLatitude       float64 `json:"pickupLatitude"`
	PickupLongitude      float64 `json:"pickupLongitude"`
	Destination          string  `json:"destination"`
	DestinationLatitude  float64 `json:"destinationLatitude"`
	DestinationLongitude float64 `json:"destinationLongitude"`
	Notes                string  `json:"notes"`
	Time                 *int64  `json:"time,omitempty"` // Nullable timestamp
	Payment              string  `json:"payment"`
	UserID               string  `json:"userId,omitempty"` // Optional field for user identification
	RouteCoordinates     string  `json:"routeCoordinates,omitempty"` // Route coordinates from OSRM (stringified array)
}

// SubmitBidRequest represents the data for submitting a bid on an order
type SubmitBidRequest struct {
	OrderID              int64   `json:"orderId"`
	DriverID             string  `json:"driverId"`
	BidPrice             float64 `json:"bidPrice"`
	EstimatedArrivalTime int64   `json:"estimatedArrivalTime"` // Timestamp waktu tiba di tempat jemput
}