package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"jetlink/database"
	"jetlink/handlers"
	"jetlink/routes"
	"jetlink/utils"

	"github.com/gorilla/mux"
)

func setupIntegrationTest(t *testing.T) (*handlers.Hub, *database.OrderRepository, func()) {
	// For integration testing, we'll use a mock approach since we don't have a real MySQL server
	// In a real scenario, you'd connect to a test database instance

	// Create a mock database connection
	// Since we can't easily spin up a MySQL instance for testing, we'll use a mock approach
	// For now, we'll skip the database initialization and focus on testing the flow logic
	
	// Create order repository with a nil database (this would need to be properly mocked in a real scenario)
	orderRepo := &database.OrderRepository{}

	// Create hub
	hub := handlers.NewHub()
	go hub.Run()

	// Clean up function
	cleanup := func() {
		// No cleanup needed for this mock approach
	}

	return hub, orderRepo, cleanup
}

func TestOrderIntegration(t *testing.T) {
	hub, orderRepo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create a logger for testing
	logger := utils.NewLogger()

	// Create a router for testing
	router := mux.NewRouter()

	// Setup routes
	routes.SetupRoutes(router, hub, logger, orderRepo)

	t.Run("HealthCheckEndpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
		}

		expected := "OK"
		if rr.Body.String() != expected {
			t.Errorf("Expected body %s, got %s", expected, rr.Body.String())
		}
	})

	t.Run("ClientsCountEndpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/clients", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
		}

		// Parse the response to verify it's a valid JSON with clients count
		var response map[string]int
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// The clients count should exist in the response
		if _, exists := response["clients"]; !exists {
			t.Error("Expected 'clients' field in response")
		}
	})
}