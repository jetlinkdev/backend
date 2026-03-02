package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"jetlink/database"
	"jetlink/firebase"
	"jetlink/models"
	"jetlink/utils"
)

// AuthHTTPHandler handles HTTP REST API requests for authentication
type AuthHTTPHandler struct {
	logger   *utils.Logger
	userRepo *database.UserRepository
}

// NewAuthHTTPHandler creates a new AuthHTTPHandler
func NewAuthHTTPHandler(logger *utils.Logger, db *database.DB) *AuthHTTPHandler {
	return &AuthHTTPHandler{
		logger:   logger,
		userRepo: database.NewUserRepository(db),
	}
}

// RegisterDriverRequest represents the request body for driver registration
type RegisterDriverRequest struct {
	VehicleType  string `json:"vehicleType"`
	VehiclePlate string `json:"vehiclePlate"`
	PhoneNumber  string `json:"phoneNumber"`
	DisplayName  string `json:"displayName"`
	Email        string `json:"email"`
}

// DriverStatusResponse represents the response for driver status check
type DriverStatusResponse struct {
	IsDriver     bool   `json:"isDriver"`
	IsVerified   bool   `json:"isVerified"`
	VehicleType  string `json:"vehicleType"`
	VehiclePlate string `json:"vehiclePlate"`
	DisplayName  string `json:"displayName"`
	Email        string `json:"email"`
	PhoneNumber  string `json:"phoneNumber"`
}

// RegisterDriver handles POST /api/auth/register-driver
// Registers a user as a driver with vehicle information
func (h *AuthHTTPHandler) RegisterDriver(w http.ResponseWriter, r *http.Request) {
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
	var req RegisterDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.VehicleType == "" {
		h.sendError(w, "Vehicle type is required", http.StatusBadRequest)
		return
	}
	if req.VehiclePlate == "" {
		h.sendError(w, "Vehicle plate is required", http.StatusBadRequest)
		return
	}
	if req.PhoneNumber == "" {
		h.sendError(w, "Phone number is required", http.StatusBadRequest)
		return
	}

	h.logger.Info(fmt.Sprintf("Registering driver: %s", firebaseUID))

	// Check if user exists
	existingUser, err := h.userRepo.GetUserByID(firebaseUID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		h.sendError(w, "Failed to register driver", http.StatusInternalServerError)
		return
	}

	if existingUser != nil {
		// User exists, check if already a driver
		if existingUser.Role == "driver" {
			h.logger.Info(fmt.Sprintf("User %s is already a driver", firebaseUID))
			// Return current status
			response := DriverStatusResponse{
				IsDriver:     true,
				IsVerified:   existingUser.IsVerified,
				VehicleType:  existingUser.VehicleType,
				VehiclePlate: existingUser.VehiclePlate,
				DisplayName:  existingUser.DisplayName,
				Email:        existingUser.Email,
				PhoneNumber:  existingUser.PhoneNumber,
			}
			h.sendSuccess(w, response, http.StatusOK)
			return
		}

		// Update existing user to driver
		existingUser.Role = "driver"
		existingUser.VehicleType = req.VehicleType
		existingUser.VehiclePlate = req.VehiclePlate
		existingUser.PhoneNumber = req.PhoneNumber
		existingUser.IsVerified = true
		existingUser.UpdatedAt = time.Now().Unix()

		if err := h.userRepo.RegisterDriver(existingUser); err != nil {
			h.logger.Error(fmt.Sprintf("Failed to update driver: %v", err))
			h.sendError(w, "Failed to register driver", http.StatusInternalServerError)
			return
		}

		h.logger.Info(fmt.Sprintf("Driver updated: %s (%s)", existingUser.Email, existingUser.VehiclePlate))

		response := DriverStatusResponse{
			IsDriver:     true,
			IsVerified:   true,
			VehicleType:  existingUser.VehicleType,
			VehiclePlate: existingUser.VehiclePlate,
			DisplayName:  existingUser.DisplayName,
			Email:        existingUser.Email,
			PhoneNumber:  existingUser.PhoneNumber,
		}
		h.sendSuccess(w, response, http.StatusOK)
		return
	}

	// User doesn't exist, create new user with driver role
	user := &models.User{
		ID:            firebaseUID,
		Email:         req.Email,
		DisplayName:   req.DisplayName,
		PhoneNumber:   req.PhoneNumber,
		Role:          "driver",
		VehicleType:   req.VehicleType,
		VehiclePlate:  req.VehiclePlate,
		IsVerified:    true,
		DriverRating:  0.0,
		TotalTrips:    0,
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}

	if err := h.userRepo.CreateUser(user); err != nil {
		h.logger.Error(fmt.Sprintf("Failed to create driver: %v", err))
		h.sendError(w, "Failed to create driver", http.StatusInternalServerError)
		return
	}

	h.logger.Info(fmt.Sprintf("New driver registered: %s (%s)", user.Email, user.VehiclePlate))

	response := DriverStatusResponse{
		IsDriver:     true,
		IsVerified:   true,
		VehicleType:  user.VehicleType,
		VehiclePlate: user.VehiclePlate,
		DisplayName:  user.DisplayName,
		Email:        user.Email,
		PhoneNumber:  user.PhoneNumber,
	}
	h.sendSuccess(w, response, http.StatusCreated)
}

// CheckDriverStatus handles GET /api/auth/driver-status
// Checks if the authenticated user is registered as a driver
func (h *AuthHTTPHandler) CheckDriverStatus(w http.ResponseWriter, r *http.Request) {
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

	// Check if user exists and is a driver
	user, err := h.userRepo.GetUserByID(firebaseUID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		h.sendError(w, "Failed to check driver status", http.StatusInternalServerError)
		return
	}

	isDriver := false
	isVerified := false
	var vehicleType, vehiclePlate, displayName, email, phoneNumber string

	if user != nil {
		isDriver = user.Role == "driver"
		isVerified = user.IsVerified
		vehicleType = user.VehicleType
		vehiclePlate = user.VehiclePlate
		displayName = user.DisplayName
		email = user.Email
		phoneNumber = user.PhoneNumber
	}

	response := DriverStatusResponse{
		IsDriver:     isDriver,
		IsVerified:   isVerified,
		VehicleType:  vehicleType,
		VehiclePlate: vehiclePlate,
		DisplayName:  displayName,
		Email:        email,
		PhoneNumber:  phoneNumber,
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// VerifyAuth handles POST /api/auth/verify
// Verifies the user's authentication and returns basic profile
func (h *AuthHTTPHandler) VerifyAuth(w http.ResponseWriter, r *http.Request) {
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

	// Check if user exists in database
	user, err := h.userRepo.GetUserByID(firebaseUID)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Failed to get user: %v", err))
		h.sendError(w, "Failed to verify user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"uid":    firebaseUID,
		"exists": user != nil,
	}

	if user != nil {
		response["role"] = user.Role
		response["isVerified"] = user.IsVerified
		response["vehicleType"] = user.VehicleType
		response["vehiclePlate"] = user.VehiclePlate
		response["phoneNumber"] = user.PhoneNumber
		response["email"] = user.Email
		response["displayName"] = user.DisplayName
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// sendSuccess sends a successful JSON response
func (h *AuthHTTPHandler) sendSuccess(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// sendError sends an error JSON response
func (h *AuthHTTPHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// FirebaseAuthMiddleware verifies Firebase ID token and adds user to context
func FirebaseAuthMiddleware(logger *utils.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Error("Missing Authorization header")
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Extract token (format: "Bearer <token>")
			var idToken string
			if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
				idToken = authHeader[7:]
			} else {
				logger.Error("Invalid Authorization header format")
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			// Verify Firebase token using Firebase Admin SDK
			ctx := r.Context()
			token, err := firebase.VerifyIDToken(ctx, idToken)
			if err != nil {
				logger.Error(fmt.Sprintf("Firebase token verification failed: %v", err))
				http.Error(w, "Unauthorized - Invalid token", http.StatusUnauthorized)
				return
			}

			// Token verified successfully, add UID to context
			ctx = context.WithValue(ctx, "firebaseUID", token.UID)
			ctx = context.WithValue(ctx, "firebaseEmail", token.Claims["email"])
			
			logger.Info(fmt.Sprintf("Token verified for user: %s", token.UID))
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
