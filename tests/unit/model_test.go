package unit

import (
	"encoding/json"
	"testing"
	"time"

	"jetlink/models"
)

func TestOrderModel(t *testing.T) {
	t.Run("CreateOrderWithValidData", func(t *testing.T) {
		now := time.Now().Unix()
		order := models.Order{
			ID:                   1,
			UserID:               "user123",
			DriverID:             "driver456",
			Pickup:               "Central Station",
			PickupLatitude:       -6.200000,
			PickupLongitude:      106.816667,
			Destination:          "Airport",
			DestinationLatitude:  -6.175383,
			DestinationLongitude: 106.643600,
			Notes:                "Please arrive 10 minutes early",
			Time:                 &now,
			Payment:              "credit_card",
			Status:               "pending",
			Fare:                 150000.00,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		if order.UserID != "user123" {
			t.Errorf("Expected UserID to be 'user123', got '%s'", order.UserID)
		}

		if order.Pickup != "Central Station" {
			t.Errorf("Expected Pickup to be 'Central Station', got '%s'", order.Pickup)
		}

		if order.Time == nil || *order.Time != now {
			t.Errorf("Expected Time to be %d, got %v", now, order.Time)
		}

		if order.Status != "pending" {
			t.Errorf("Expected Status to be 'pending', got '%s'", order.Status)
		}
	})

	t.Run("CreateOrderWithNullTime", func(t *testing.T) {
		order := models.Order{
			ID:                   2,
			UserID:               "user123",
			Pickup:               "Central Station",
			PickupLatitude:       -6.200000,
			PickupLongitude:      106.816667,
			Destination:          "Airport",
			DestinationLatitude:  -6.175383,
			DestinationLongitude: 106.643600,
			Notes:                "No special instructions",
			Time:                 nil, // Nil time means "as soon as possible"
			Payment:              "cash",
			Status:               "pending",
			Fare:                 120000.00,
			CreatedAt:            time.Now().Unix(),
			UpdatedAt:            time.Now().Unix(),
		}

		if order.Time != nil {
			t.Errorf("Expected Time to be nil, got %v", *order.Time)
		}
	})

	t.Run("CreateOrderRequestWithValidData", func(t *testing.T) {
		now := time.Now().Unix()
		req := models.CreateOrderRequest{
			Pickup:               "Hotel ABC",
			PickupLatitude:       -6.175383,
			PickupLongitude:      106.827870,
			Destination:          "Shopping Mall",
			DestinationLatitude:  -6.227480,
			DestinationLongitude: 106.805220,
			Notes:                "Ring the bell at entrance",
			Time:                 &now,
			Payment:              "ewallet",
			UserID:               "user789",
		}

		if req.Pickup != "Hotel ABC" {
			t.Errorf("Expected Pickup to be 'Hotel ABC', got '%s'", req.Pickup)
		}

		if req.Time == nil || *req.Time != now {
			t.Errorf("Expected Time to be %d, got %v", now, req.Time)
		}

		if req.UserID != "user789" {
			t.Errorf("Expected UserID to be 'user789', got '%s'", req.UserID)
		}
	})

	t.Run("CreateOrderRequestWithNullTime", func(t *testing.T) {
		req := models.CreateOrderRequest{
			Pickup:               "Restaurant XYZ",
			PickupLatitude:       -6.214620,
			PickupLongitude:      106.845130,
			Destination:          "Office Park",
			DestinationLatitude:  -6.230370,
			DestinationLongitude: 106.823000,
			Notes:                "Call when arriving",
			Time:                 nil, // Nil time means "as soon as possible"
			Payment:              "credit_card",
			UserID:               "user101",
		}

		if req.Time != nil {
			t.Errorf("Expected Time to be nil, got %v", req.Time)
		}
	})

	t.Run("OrderModelJSONSerialization", func(t *testing.T) {
		now := time.Now().Unix()
		order := models.Order{
			ID:                   3,
			UserID:               "user123",
			Pickup:               "Central Station",
			PickupLatitude:       -6.200000,
			PickupLongitude:      106.816667,
			Destination:          "Airport",
			DestinationLatitude:  -6.175383,
			DestinationLongitude: 106.643600,
			Notes:                "Please arrive 10 minutes early",
			Time:                 &now,
			Payment:              "credit_card",
			Status:               "pending",
			Fare:                 150000.00,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		jsonData, err := json.Marshal(order)
		if err != nil {
			t.Fatalf("Failed to marshal order to JSON: %v", err)
		}

		var unmarshaledOrder models.Order
		err = json.Unmarshal(jsonData, &unmarshaledOrder)
		if err != nil {
			t.Fatalf("Failed to unmarshal order from JSON: %v", err)
		}

		if unmarshaledOrder.UserID != order.UserID {
			t.Errorf("UserID mismatch after serialization/deserialization: expected %s, got %s", order.UserID, unmarshaledOrder.UserID)
		}

		if unmarshaledOrder.Pickup != order.Pickup {
			t.Errorf("Pickup mismatch after serialization/deserialization: expected %s, got %s", order.Pickup, unmarshaledOrder.Pickup)
		}

		if unmarshaledOrder.Time == nil || *unmarshaledOrder.Time != *order.Time {
			t.Errorf("Time mismatch after serialization/deserialization: expected %d, got %v", *order.Time, unmarshaledOrder.Time)
		}
	})

	t.Run("OrderModelJSONSerializationWithNullTime", func(t *testing.T) {
		now := time.Now().Unix()
		order := models.Order{
			ID:                   4,
			UserID:               "user123",
			Pickup:               "Central Station",
			PickupLatitude:       -6.200000,
			PickupLongitude:      106.816667,
			Destination:          "Airport",
			DestinationLatitude:  -6.175383,
			DestinationLongitude: 106.643600,
			Notes:                "Please arrive 10 minutes early",
			Time:                 nil, // Nil time
			Payment:              "credit_card",
			Status:               "pending",
			Fare:                 150000.00,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		jsonData, err := json.Marshal(order)
		if err != nil {
			t.Fatalf("Failed to marshal order to JSON: %v", err)
		}

		var unmarshaledOrder models.Order
		err = json.Unmarshal(jsonData, &unmarshaledOrder)
		if err != nil {
			t.Fatalf("Failed to unmarshal order from JSON: %v", err)
		}

		if unmarshaledOrder.Time != nil {
			t.Errorf("Time should be nil after serialization/deserialization, got %v", unmarshaledOrder.Time)
		}
	})
}