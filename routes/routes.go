package routes

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	hubhandlers "jetlink/handlers"
	"jetlink/database"
	"jetlink/utils"
	"jetlink/websocket"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(router *mux.Router, hub *hubhandlers.Hub, logger *utils.Logger, repo *database.OrderRepository) {
	// Create HTTP auth handler
	authHandler := hubhandlers.NewAuthHTTPHandler(logger, repo.GetDB())

	// Create HTTP bid handler
	bidHandler := hubhandlers.NewBidHTTPHandler(logger, repo.GetDB())

	// Create Firebase auth middleware
	authMiddleware := hubhandlers.FirebaseAuthMiddleware(logger)

	// REST API Routes - Authentication
	router.HandleFunc("/api/auth/register-driver", authHandler.RegisterDriver).Methods("POST", "OPTIONS")
	router.Handle("/api/auth/driver-status", authMiddleware(http.HandlerFunc(authHandler.CheckDriverStatus))).Methods("GET", "OPTIONS")
	router.Handle("/api/auth/verify", authMiddleware(http.HandlerFunc(authHandler.VerifyAuth))).Methods("POST", "OPTIONS")

	// REST API Routes - Bids
	router.Handle("/api/bids/submit", authMiddleware(http.HandlerFunc(bidHandler.SubmitBid))).Methods("POST", "OPTIONS")
	router.Handle("/api/bids/my", authMiddleware(http.HandlerFunc(bidHandler.GetMyBids))).Methods("GET", "OPTIONS")
	router.Handle("/api/bids/order/{orderId}", authMiddleware(http.HandlerFunc(bidHandler.GetOrderBids))).Methods("GET", "OPTIONS")

	// WebSocket endpoint
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ConnectionsHandler(w, r, hub, logger, repo)
	})

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Endpoint to get client count
	router.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		count := hub.GetClientsCount()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"clients": %d}`, count)
	})
}