package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"jetlink/models"
)

// OrderRedis handles order-related Redis operations
type OrderRedis struct {
	client *Client
}

// OrderTTL is the time-to-live for active orders in Redis
const OrderTTL = 30 * time.Minute

// NewOrderRedis creates a new OrderRedis instance
func NewOrderRedis(client *Client) *OrderRedis {
	return &OrderRedis{
		client: client,
	}
}

// CreateActiveOrder stores an active order in Redis
func (r *OrderRedis) CreateActiveOrder(ctx context.Context, order *models.Order, clientID string) error {
	// Store order data as JSON
	key := fmt.Sprintf("order:active:%d", order.ID)
	
	data := map[string]interface{}{
		"id":                      order.ID,
		"user_id":                 order.UserID,
		"driver_id":               order.DriverID,
		"pickup":                  order.Pickup,
		"pickup_latitude":         order.PickupLatitude,
		"pickup_longitude":        order.PickupLongitude,
		"destination":             order.Destination,
		"destination_latitude":    order.DestinationLatitude,
		"destination_longitude":   order.DestinationLongitude,
		"notes":                   order.Notes,
		"payment":                 order.Payment,
		"status":                  order.Status,
		"fare":                    order.Fare,
		"bid_price":               order.BidPrice,
		"created_at":              order.CreatedAt,
		"updated_at":              order.UpdatedAt,
		"client_id":               clientID,
	}
	
	// Handle nullable fields
	if order.Time != nil {
		data["time"] = *order.Time
	}
	if order.EstimatedArrivalTime != nil {
		data["estimated_arrival_time"] = *order.EstimatedArrivalTime
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %v", err)
	}

	// Store order JSON
	err = r.client.InnerClient().Set(ctx, key, string(jsonData), OrderTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to store order: %v", err)
	}

	// Create client -> order mapping
	clientOrderKey := fmt.Sprintf("client:order:%s", clientID)
	err = r.client.InnerClient().Set(ctx, clientOrderKey, order.ID, OrderTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to store client-order mapping: %v", err)
	}

	// Create order -> client mapping
	orderClientKey := fmt.Sprintf("order:client:%d", order.ID)
	err = r.client.InnerClient().Set(ctx, orderClientKey, clientID, OrderTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to store order-client mapping: %v", err)
	}

	// Add to user's active orders set
	userOrdersKey := fmt.Sprintf("user:orders:%s", order.UserID)
	err = r.client.InnerClient().SAdd(ctx, userOrdersKey, order.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to add to user orders: %v", err)
	}
	r.client.InnerClient().Expire(ctx, userOrdersKey, OrderTTL)

	// Add to available orders for drivers (if pending)
	if order.Status == "pending" {
		availableOrdersKey := "orders:available"
		err = r.client.InnerClient().ZAdd(ctx, availableOrdersKey, &redis.Z{
			Score:  float64(order.CreatedAt),
			Member: order.ID,
		}).Err()
		if err != nil {
			return fmt.Errorf("failed to add to available orders: %v", err)
		}
		r.client.InnerClient().Expire(ctx, availableOrdersKey, OrderTTL)
	}

	return nil
}

// GetOrderByID retrieves an order by ID from Redis
func (r *OrderRedis) GetOrderByID(ctx context.Context, orderID int64) (*models.Order, error) {
	key := fmt.Sprintf("order:active:%d", orderID)

	data, err := r.client.InnerClient().Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Order not found/expired
		}
		return nil, fmt.Errorf("failed to get order: %v", err)
	}

	var order models.Order
	err = json.Unmarshal([]byte(data), &order)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal order: %v", err)
	}

	return &order, nil
}

// GetOrderByClientID retrieves the active order for a client
func (r *OrderRedis) GetOrderByClientID(ctx context.Context, clientID string) (*models.Order, error) {
	// Get order ID from client
	clientOrderKey := fmt.Sprintf("client:order:%s", clientID)
	orderIDStr, err := r.client.InnerClient().Get(ctx, clientOrderKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No active order for this client
		}
		return nil, fmt.Errorf("failed to get client order: %v", err)
	}

	var orderID int64
	fmt.Sscanf(orderIDStr, "%d", &orderID)

	// Get order data
	return r.GetOrderByID(ctx, orderID)
}

// GetClientIDByOrderID retrieves the client ID for an order
func (r *OrderRedis) GetClientIDByOrderID(ctx context.Context, orderID int64) (string, error) {
	key := fmt.Sprintf("order:client:%d", orderID)

	clientID, err := r.client.InnerClient().Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // No client mapped to this order
		}
		return "", fmt.Errorf("failed to get client ID: %v", err)
	}

	return clientID, nil
}

// UpdateOrderStatus updates the status of an order in Redis
func (r *OrderRedis) UpdateOrderStatus(ctx context.Context, orderID int64, status string) error {
	// Get existing order
	order, err := r.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("order not found")
	}

	// Update status
	order.Status = status
	order.UpdatedAt = time.Now().Unix()

	// Store updated order
	key := fmt.Sprintf("order:active:%d", orderID)
	jsonData, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %v", err)
	}

	err = r.client.InnerClient().Set(ctx, key, string(jsonData), OrderTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to update order: %v", err)
	}

	// Remove from available orders if no longer pending
	if status != "pending" {
		availableOrdersKey := "orders:available"
		r.client.InnerClient().ZRem(ctx, availableOrdersKey, orderID)
	}

	return nil
}

// DeleteActiveOrder removes an order from Redis (after completion/cancellation)
func (r *OrderRedis) DeleteActiveOrder(ctx context.Context, orderID int64, clientID string) error {
	// Get order to find user_id
	order, err := r.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Delete order data
	key := fmt.Sprintf("order:active:%d", orderID)
	r.client.InnerClient().Del(ctx, key)

	// Delete client-order mapping
	clientOrderKey := fmt.Sprintf("client:order:%s", clientID)
	r.client.InnerClient().Del(ctx, clientOrderKey)

	// Delete order-client mapping
	orderClientKey := fmt.Sprintf("order:client:%d", orderID)
	r.client.InnerClient().Del(ctx, orderClientKey)

	// Remove from user's orders set
	if order != nil {
		userOrdersKey := fmt.Sprintf("user:orders:%s", order.UserID)
		r.client.InnerClient().SRem(ctx, userOrdersKey, orderID)
	}

	// Remove from available orders
	availableOrdersKey := "orders:available"
	r.client.InnerClient().ZRem(ctx, availableOrdersKey, orderID)

	return nil
}

// GetUserActiveOrders retrieves all active orders for a user
func (r *OrderRedis) GetUserActiveOrders(ctx context.Context, userID string) ([]int64, error) {
	key := fmt.Sprintf("user:orders:%s", userID)

	orderIDs, err := r.client.InnerClient().SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %v", err)
	}

	var orders []int64
	for _, idStr := range orderIDs {
		var id int64
		fmt.Sscanf(idStr, "%d", &id)
		orders = append(orders, id)
	}

	return orders, nil
}

// GetAvailableOrders retrieves all pending orders available for drivers
func (r *OrderRedis) GetAvailableOrders(ctx context.Context) ([]int64, error) {
	key := "orders:available"

	orderIDs, err := r.client.InnerClient().ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get available orders: %v", err)
	}

	var orders []int64
	for _, idStr := range orderIDs {
		var id int64
		fmt.Sscanf(idStr, "%d", &id)
		orders = append(orders, id)
	}

	return orders, nil
}

// RefreshOrderTTL refreshes the TTL of an order
func (r *OrderRedis) RefreshOrderTTL(ctx context.Context, orderID int64) error {
	key := fmt.Sprintf("order:active:%d", orderID)
	return r.client.InnerClient().Expire(ctx, key, OrderTTL).Err()
}

// GetClient returns the Redis client
func (r *OrderRedis) GetClient() *Client {
	return r.client
}
