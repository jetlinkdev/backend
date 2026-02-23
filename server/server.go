package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"jetlink/utils"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	logger     *utils.Logger
}

// New creates a new server instance
func New(addr string, router *mux.Router, logger *utils.Logger) *Server {
	// Enable CORS
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handlers.CORS(originsOk, headersOk, methodsOk)(router),
		ReadTimeout:  60 * time.Second,  // Increased for WebSocket connections
		WriteTimeout: 60 * time.Second,  // Increased for WebSocket connections
		IdleTimeout:  120 * time.Second, // Allow idle connections for WebSocket
	}

	return &Server{
		httpServer: httpServer,
		logger:     logger,
	}
}

// Start starts the HTTP server
func (s *Server) Start() {
	// Start server in a goroutine
	go func() {
		s.logger.Info(fmt.Sprintf("Server starting on %s", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error(fmt.Sprintf("Server error: %v", err))
		}
	}()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.logger.Info("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error(fmt.Sprintf("Server forced to shutdown: %v", err))
	} else {
		s.logger.Info("Server exited properly")
	}
}

// WaitForShutdownSignal waits for interrupt signal to gracefully shutdown the server
func (s *Server) WaitForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}