package models

// Review represents a user's review for a driver
type Review struct {
	ID        int64  `json:"id"`
	OrderID   int64  `json:"orderId"`
	UserID    string `json:"userId"`
	DriverID  string `json:"driverId"`
	Rating    int    `json:"rating"`
	Review    string `json:"review,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// CreateReviewRequest represents the data for creating a review
type CreateReviewRequest struct {
	OrderID  int64  `json:"orderId"`
	DriverID string `json:"driverId"`
	Rating   int    `json:"rating"`
	Review   string `json:"review,omitempty"`
}

// UpdateDriverRatingRequest represents the data for updating driver's average rating
type UpdateDriverRatingRequest struct {
	DriverID string  `json:"driverId"`
	Rating   float64 `json:"rating"`
}
