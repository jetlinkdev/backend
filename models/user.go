package models

// User represents a user in the system (customer or driver)
type User struct {
	ID              string  `json:"id"`                  // Firebase UID
	Email           string  `json:"email"`
	DisplayName     string  `json:"displayName,omitempty"`
	PhotoURL        string  `json:"photoUrl,omitempty"`
	PhoneNumber     string  `json:"phoneNumber,omitempty"`
	Role            string  `json:"role"` // "customer" or "driver"
	VehicleType     string  `json:"vehicleType,omitempty"`
	VehiclePlate    string  `json:"vehiclePlate,omitempty"`
	DriverRating    float64 `json:"driverRating,omitempty"`
	TotalTrips      int     `json:"totalTrips,omitempty"`
	IsVerified      bool    `json:"isVerified"`
	CreatedAt       int64   `json:"createdAt"`
	UpdatedAt       int64   `json:"updatedAt"`
	LastLoginAt     *int64  `json:"lastLoginAt,omitempty"`
}

// CreateUserRequest represents the data for creating a new user
type CreateUserRequest struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"displayName,omitempty"`
	PhotoURL      string `json:"photoUrl,omitempty"`
	PhoneNumber   string `json:"phoneNumber,omitempty"`
	Role          string `json:"role"`
}

// UpdateUserRequest represents the data for updating a user
type UpdateUserRequest struct {
	DisplayName   string `json:"displayName,omitempty"`
	PhoneNumber   string `json:"phoneNumber,omitempty"`
	VehicleType   string `json:"vehicleType,omitempty"`
	VehiclePlate  string `json:"vehiclePlate,omitempty"`
}

// DriverRegistrationRequest represents the data for driver registration
type DriverRegistrationRequest struct {
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	PhoneNumber  string `json:"phoneNumber"`
	VehicleType  string `json:"vehicleType"`
	VehiclePlate string `json:"vehiclePlate"`
}
