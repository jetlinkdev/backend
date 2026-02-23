package unit

import (
	"errors"
	"testing"
	"time"

	"jetlink/models"
)

// MockOrderRepository untuk keperluan testing
type MockOrderRepository struct {
	orders map[int64]*models.Order
	nextID int64
}

func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{
		orders: make(map[int64]*models.Order),
		nextID: 1,
	}
}

func (m *MockOrderRepository) CreateOrder(order *models.Order) error {
	order.ID = m.nextID
	m.nextID++
	m.orders[order.ID] = order
	return nil
}

func (m *MockOrderRepository) GetOrder(id int64) (*models.Order, error) {
	order, exists := m.orders[id]
	if !exists {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (m *MockOrderRepository) UpdateOrder(order *models.Order) error {
	_, exists := m.orders[order.ID]
	if !exists {
		return errors.New("order not found")
	}
	m.orders[order.ID] = order
	return nil
}

func (m *MockOrderRepository) GetOrdersByUserID(userID string) ([]*models.Order, error) {
	var orders []*models.Order
	for _, order := range m.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (m *MockOrderRepository) GetOrdersByStatus(status string) ([]*models.Order, error) {
	var orders []*models.Order
	for _, order := range m.orders {
		if order.Status == status {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (m *MockOrderRepository) GetAllOrders() ([]*models.Order, error) {
	var orders []*models.Order
	for _, order := range m.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func TestMockOrderRepository(t *testing.T) {
	t.Run("CreateAndRetrieveOrder", func(t *testing.T) {
		mockRepo := NewMockOrderRepository()
		now := time.Now().Unix()
		order := &models.Order{
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

		// Create the order
		err := mockRepo.CreateOrder(order)
		if err != nil {
			t.Fatalf("Failed to create order: %v", err)
		}

		// Check that ID was assigned
		if order.ID == 0 {
			t.Error("Expected order ID to be assigned after creation")
		}

		// Retrieve the order
		retrievedOrder, err := mockRepo.GetOrder(order.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve order: %v", err)
		}

		// Verify the retrieved order matches the created one
		if retrievedOrder.UserID != order.UserID {
			t.Errorf("UserID mismatch: expected %s, got %s", order.UserID, retrievedOrder.UserID)
		}

		if retrievedOrder.Pickup != order.Pickup {
			t.Errorf("Pickup mismatch: expected %s, got %s", order.Pickup, retrievedOrder.Pickup)
		}

		if retrievedOrder.Time == nil || *retrievedOrder.Time != *order.Time {
			t.Errorf("Time mismatch: expected %d, got %v", *order.Time, retrievedOrder.Time)
		}

		if retrievedOrder.Status != order.Status {
			t.Errorf("Status mismatch: expected %s, got %s", order.Status, retrievedOrder.Status)
		}
	})

	t.Run("CreateOrderWithNullTime", func(t *testing.T) {
		mockRepo := NewMockOrderRepository()
		now := time.Now().Unix()
		order := &models.Order{
			UserID:               "user456",
			Pickup:               "Hotel ABC",
			PickupLatitude:       -6.175383,
			PickupLongitude:      106.827870,
			Destination:          "Shopping Mall",
			DestinationLatitude:  -6.227480,
			DestinationLongitude: 106.805220,
			Notes:                "No special instructions",
			Time:                 nil, // Null time
			Payment:              "cash",
			Status:               "pending",
			Fare:                 120000.00,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		// Create the order
		err := mockRepo.CreateOrder(order)
		if err != nil {
			t.Fatalf("Failed to create order with null time: %v", err)
		}

		// Check that ID was assigned
		if order.ID == 0 {
			t.Error("Expected order ID to be assigned after creation")
		}

		// Retrieve the order
		retrievedOrder, err := mockRepo.GetOrder(order.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve order: %v", err)
		}

		// Verify the retrieved order has null time
		if retrievedOrder.Time != nil {
			t.Errorf("Expected Time to be nil, got %v", *retrievedOrder.Time)
		}
	})

	t.Run("UpdateOrder", func(t *testing.T) {
		mockRepo := NewMockOrderRepository()
		now := time.Now().Unix()
		order := &models.Order{
			UserID:               "user789",
			Pickup:               "Restaurant XYZ",
			PickupLatitude:       -6.214620,
			PickupLongitude:      106.845130,
			Destination:          "Office Park",
			DestinationLatitude:  -6.230370,
			DestinationLongitude: 106.823000,
			Notes:                "Call when arriving",
			Time:                 &now,
			Payment:              "ewallet",
			Status:               "pending",
			Fare:                 100000.00,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		// Create the order
		err := mockRepo.CreateOrder(order)
		if err != nil {
			t.Fatalf("Failed to create order: %v", err)
		}

		// Update the order
		order.Status = "accepted"
		order.DriverID = "driver123"
		newTime := time.Now().Unix()
		order.Time = &newTime
		order.UpdatedAt = time.Now().Unix()

		err = mockRepo.UpdateOrder(order)
		if err != nil {
			t.Fatalf("Failed to update order: %v", err)
		}

		// Retrieve the updated order
		updatedOrder, err := mockRepo.GetOrder(order.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated order: %v", err)
		}

		// Verify the updates
		if updatedOrder.Status != "accepted" {
			t.Errorf("Expected status to be 'accepted', got '%s'", updatedOrder.Status)
		}

		if updatedOrder.DriverID != "driver123" {
			t.Errorf("Expected DriverID to be 'driver123', got '%s'", updatedOrder.DriverID)
		}

		if updatedOrder.Time == nil || *updatedOrder.Time != newTime {
			t.Errorf("Expected Time to be %d, got %v", newTime, updatedOrder.Time)
		}
	})

	t.Run("GetOrdersByUserID", func(t *testing.T) {
		mockRepo := NewMockOrderRepository()
		userID := "user999"
		now := time.Now().Unix()

		// Create multiple orders for the same user
		orders := []*models.Order{
			{
				UserID:               userID,
				Pickup:               "Location A",
				PickupLatitude:       -6.200000,
				PickupLongitude:      106.816667,
				Destination:          "Location B",
				DestinationLatitude:  -6.175383,
				DestinationLongitude: 106.643600,
				Notes:                "Order 1",
				Time:                 &now,
				Payment:              "credit_card",
				Status:               "completed",
				Fare:                 150000.00,
				CreatedAt:            now,
				UpdatedAt:            now,
			},
			{
				UserID:               userID,
				Pickup:               "Location C",
				PickupLatitude:       -6.175383,
				PickupLongitude:      106.827870,
				Destination:          "Location D",
				DestinationLatitude:  -6.227480,
				DestinationLongitude: 106.805220,
				Notes:                "Order 2",
				Time:                 nil, // Null time
				Payment:              "cash",
				Status:               "pending",
				Fare:                 120000.00,
				CreatedAt:            now,
				UpdatedAt:            now,
			},
		}

		for _, order := range orders {
			err := mockRepo.CreateOrder(order)
			if err != nil {
				t.Fatalf("Failed to create order: %v", err)
			}
		}

		// Retrieve orders by user ID
		retrievedOrders, err := mockRepo.GetOrdersByUserID(userID)
		if err != nil {
			t.Fatalf("Failed to retrieve orders by user ID: %v", err)
		}

		// Verify we got the right number of orders
		if len(retrievedOrders) != 2 {
			t.Errorf("Expected 2 orders, got %d", len(retrievedOrders))
		}

		// Verify the orders belong to the correct user
		for _, retrievedOrder := range retrievedOrders {
			if retrievedOrder.UserID != userID {
				t.Errorf("Expected UserID to be '%s', got '%s'", userID, retrievedOrder.UserID)
			}
		}
	})

	t.Run("GetOrdersByStatus", func(t *testing.T) {
		mockRepo := NewMockOrderRepository()
		status := "pending"
		now := time.Now().Unix()

		// Create multiple orders with the same status
		orders := []*models.Order{
			{
				UserID:               "user111",
				Pickup:               "Location E",
				PickupLatitude:       -6.200000,
				PickupLongitude:      106.816667,
				Destination:          "Location F",
				DestinationLatitude:  -6.175383,
				DestinationLongitude: 106.643600,
				Notes:                "Pending Order 1",
				Time:                 &now,
				Payment:              "credit_card",
				Status:               status,
				Fare:                 150000.00,
				CreatedAt:            now,
				UpdatedAt:            now,
			},
			{
				UserID:               "user222",
				Pickup:               "Location G",
				PickupLatitude:       -6.175383,
				PickupLongitude:      106.827870,
				Destination:          "Location H",
				DestinationLatitude:  -6.227480,
				DestinationLongitude: 106.805220,
				Notes:                "Pending Order 2",
				Time:                 nil, // Null time
				Payment:              "cash",
				Status:               status,
				Fare:                 120000.00,
				CreatedAt:            now,
				UpdatedAt:            now,
			},
		}

		for _, order := range orders {
			err := mockRepo.CreateOrder(order)
			if err != nil {
				t.Fatalf("Failed to create order: %v", err)
			}
		}

		// Also create an order with a different status to ensure filtering works
		differentStatusOrder := &models.Order{
			UserID:               "user333",
			Pickup:               "Location I",
			PickupLatitude:       -6.214620,
			PickupLongitude:      106.845130,
			Destination:          "Location J",
			DestinationLatitude:  -6.230370,
			DestinationLongitude: 106.823000,
			Notes:                "Completed Order",
			Time:                 &now,
			Payment:              "ewallet",
			Status:               "completed",
			Fare:                 100000.00,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		err := mockRepo.CreateOrder(differentStatusOrder)
		if err != nil {
			t.Fatalf("Failed to create order: %v", err)
		}

		// Retrieve orders by status
		retrievedOrders, err := mockRepo.GetOrdersByStatus(status)
		if err != nil {
			t.Fatalf("Failed to retrieve orders by status: %v", err)
		}

		// Verify we got the right number of orders
		if len(retrievedOrders) != 2 {
			t.Errorf("Expected 2 orders with status '%s', got %d", status, len(retrievedOrders))
		}

		// Verify all retrieved orders have the correct status
		for _, retrievedOrder := range retrievedOrders {
			if retrievedOrder.Status != status {
				t.Errorf("Expected Status to be '%s', got '%s'", status, retrievedOrder.Status)
			}
		}
	})

	t.Run("GetAllOrders", func(t *testing.T) {
		mockRepo := NewMockOrderRepository()
		now := time.Now().Unix()

		// Count existing orders
		existingOrders, err := mockRepo.GetAllOrders()
		if err != nil {
			t.Fatalf("Failed to get existing orders: %v", err)
		}
		initialCount := len(existingOrders)

		// Create a new order
		order := &models.Order{
			UserID:               "user555",
			Pickup:               "Location K",
			PickupLatitude:       -6.200000,
			PickupLongitude:      106.816667,
			Destination:          "Location L",
			DestinationLatitude:  -6.175383,
			DestinationLongitude: 106.643600,
			Notes:                "All Orders Test",
			Time:                 &now,
			Payment:              "credit_card",
			Status:               "pending",
			Fare:                 150000.00,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		err = mockRepo.CreateOrder(order)
		if err != nil {
			t.Fatalf("Failed to create order: %v", err)
		}

		// Get all orders again
		allOrders, err := mockRepo.GetAllOrders()
		if err != nil {
			t.Fatalf("Failed to get all orders: %v", err)
		}

		// Verify the count increased by 1
		if len(allOrders) != initialCount+1 {
			t.Errorf("Expected %d orders, got %d", initialCount+1, len(allOrders))
		}

		// Verify the newly created order is in the list
		found := false
		for _, retrievedOrder := range allOrders {
			if retrievedOrder.ID == order.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Newly created order was not found in GetAllOrders result")
		}
	})

	t.Run("GetNonExistentOrder", func(t *testing.T) {
		mockRepo := NewMockOrderRepository()
		// Try to retrieve an order that doesn't exist
		_, err := mockRepo.GetOrder(999999) // Very high ID that shouldn't exist
		if err == nil {
			t.Error("Expected an error when retrieving non-existent order, but got none")
		}
	})
}