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

	// REST API Routes - Authentication
	router.HandleFunc("/api/auth/register-driver", authHandler.RegisterDriver).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/driver-status", authHandler.CheckDriverStatus).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/auth/verify", authHandler.VerifyAuth).Methods("POST", "OPTIONS")

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