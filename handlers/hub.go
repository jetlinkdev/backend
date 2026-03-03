package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"jetlink/redis"
)

// Client represents a connected WebSocket client
type Client struct {
	ID           string
	Conn         *websocket.Conn
	Send         chan []byte
	OrderID      *int64  // Current active order ID (stored in Redis, cached here)
	UserID       string  // User identifier (Firebase UID)
	Role         string  // "customer" | "driver"
	DriverStatus string  // "available" | "busy" | "offline" (for drivers only)
	mu           sync.Mutex // Mutex to prevent concurrent close
	closed       bool
}

// UserOrderState represents the current order state for a user
type UserOrderState struct {
	OrderID       int64
	Status        string // pending, accepted, in_progress, completed, cancelled
	UIState       string // booking, waiting_bids, driver_assigned, completed, cancelled
	CreatedAt     int64
	LastUpdatedAt int64
}

// Hub manages all connected clients and broadcasts messages
type Hub struct {
	// Registered clients
	Clients map[*Client]bool

	// Messages to be sent to clients
	Broadcast chan []byte

	// Register new client
	Register chan *Client

	// Unregister client
	Unregister chan *Client

	// Mutex for thread-safe operations
	Mu sync.RWMutex

	// Track connections per user (userID -> set of clients)
	UserConnections map[string]map[*Client]bool

	// Track order state per user (userID -> order state)
	UserOrders map[string]*UserOrderState

	// Redis client for order storage
	OrderRedis *redis.OrderRedis
	BidRedis   *redis.BidRedis
}

// NewHub creates a new hub instance
func NewHub() *Hub {
	return &Hub{
		Broadcast:       make(chan []byte),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		Clients:         make(map[*Client]bool),
		UserConnections: make(map[string]map[*Client]bool),
		UserOrders:      make(map[string]*UserOrderState),
	}
}

// NewHubWithRedis creates a new hub instance with Redis repositories
func NewHubWithRedis(orderRedis *redis.OrderRedis, bidRedis *redis.BidRedis) *Hub {
	return &Hub{
		Broadcast:       make(chan []byte),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		Clients:         make(map[*Client]bool),
		UserConnections: make(map[string]map[*Client]bool),
		UserOrders:      make(map[string]*UserOrderState),
		OrderRedis:      orderRedis,
		BidRedis:        bidRedis,
	}
}

// Run manages the hub's register/unregister/broadcast operations
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mu.Lock()
			h.Clients[client] = true
			h.Mu.Unlock()
			log.Printf("Client registered. Total clients: %d", len(h.Clients))
			
		case client := <-h.Unregister:
			h.Mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				log.Printf("Client unregistered. Total clients: %d", len(h.Clients))
			}
			h.Mu.Unlock()
			
		case message := <-h.Broadcast:
			h.Mu.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.Mu.RUnlock()
		}
	}
}

// Message represents a WebSocket message in the format: {"intent": "...", "data": {...}}
type Message struct {
	Intent    string      `json:"intent"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	ClientID  string      `json:"clientId,omitempty"`
}

// BroadcastMessage sends a message to all connected clients
func (h *Hub) BroadcastMessage(msg Message) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	
	for client := range h.Clients {
		client.Send <- msg.ToJSON()
	}
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, msg Message) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	
	for client := range h.Clients {
		if client.ID == clientID {
			client.Send <- msg.ToJSON()
			break
		}
	}
}

// ToJSON converts a message to JSON bytes
func (m Message) ToJSON() []byte {
	data, err := json.Marshal(m)
	if err != nil {
		log.Printf("Error marshaling message to JSON: %v", err)
		return []byte{}
	}
	return data
}

// GetClientsCount returns the number of connected clients
func (h *Hub) GetClientsCount() int {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	return len(h.Clients)
}

// SetClientOrder sets the order ID for a client and stores it in Redis
func (h *Hub) SetClientOrder(client *Client, orderID int64) error {
	h.Mu.Lock()
	client.OrderID = &orderID
	h.Mu.Unlock()

	// Store in Redis if available
	if h.OrderRedis != nil {
		ctx := context.Background()
		clientID := client.ID
		
		// Store client -> order mapping
		err := h.OrderRedis.GetClient().InnerClient().Set(
			ctx,
			fmt.Sprintf("client:order:%s", clientID),
			orderID,
			redis.OrderTTL,
		).Err()
		if err != nil {
			log.Printf("Failed to store client-order mapping in Redis: %v", err)
		}

		// Store order -> client mapping
		err = h.OrderRedis.GetClient().InnerClient().Set(
			ctx,
			fmt.Sprintf("order:client:%d", orderID),
			clientID,
			redis.OrderTTL,
		).Err()
		if err != nil {
			log.Printf("Failed to store order-client mapping in Redis: %v", err)
		}
	}

	return nil
}

// GetClientByOrderID finds a client by order ID
func (h *Hub) GetClientByOrderID(orderID int64) *Client {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	for client := range h.Clients {
		if client.OrderID != nil && *client.OrderID == orderID {
			return client
		}
	}
	return nil
}

// ClearClientOrder clears the order ID for a client and removes it from Redis
func (h *Hub) ClearClientOrder(client *Client) {
	h.Mu.Lock()
	client.OrderID = nil
	h.Mu.Unlock()

	// Remove from Redis if available
	if h.OrderRedis != nil {
		ctx := context.Background()
		clientID := client.ID

		// Get order ID before clearing
		orderIDStr, _ := h.OrderRedis.GetClient().InnerClient().Get(
			ctx,
			fmt.Sprintf("client:order:%s", clientID),
		).Result()

		// Delete mappings
		h.OrderRedis.GetClient().InnerClient().Del(
			ctx,
			fmt.Sprintf("client:order:%s", clientID),
		)

		if orderIDStr != "" {
			h.OrderRedis.GetClient().InnerClient().Del(
				ctx,
				fmt.Sprintf("order:client:%s", orderIDStr),
			)
		}
	}
}

// AssociateClientWithUser links a client connection to a user ID
func (h *Hub) AssociateClientWithUser(client *Client, userID string) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if h.UserConnections == nil {
		h.UserConnections = make(map[string]map[*Client]bool)
	}

	if h.UserConnections[userID] == nil {
		h.UserConnections[userID] = make(map[*Client]bool)
	}

	h.UserConnections[userID][client] = true
	client.UserID = userID

	log.Printf("Client %s associated with user %s. Total connections for user: %d",
		client.ID, userID, len(h.UserConnections[userID]))
}

// RemoveClientFromUser removes a client from user's connections (called on disconnect)
func (h *Hub) RemoveClientFromUser(client *Client) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if client.UserID == "" {
		return
	}

	if h.UserConnections[client.UserID] != nil {
		delete(h.UserConnections[client.UserID], client)

		// If no more connections for this user, cleanup
		if len(h.UserConnections[client.UserID]) == 0 {
			delete(h.UserConnections, client.UserID)
			log.Printf("User %s has no more active connections", client.UserID)
		}
	}
}

// SetUserOrderState updates or creates order state for a user
func (h *Hub) SetUserOrderState(userID string, orderID int64, status string, uiState string) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if h.UserOrders == nil {
		h.UserOrders = make(map[string]*UserOrderState)
	}

	h.UserOrders[userID] = &UserOrderState{
		OrderID:       orderID,
		Status:        status,
		UIState:       uiState,
		CreatedAt:     time.Now().Unix(),
		LastUpdatedAt: time.Now().Unix(),
	}

	log.Printf("Order state set for user %s: order=%d, status=%s, ui=%s",
		userID, orderID, status, uiState)
}

// GetUserOrderState retrieves order state for a user
func (h *Hub) GetUserOrderState(userID string) *UserOrderState {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	if h.UserOrders == nil {
		return nil
	}

	return h.UserOrders[userID]
}

// ClearUserOrderState clears order state for a user (after cancel/complete)
func (h *Hub) ClearUserOrderState(userID string) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if h.UserOrders != nil {
		delete(h.UserOrders, userID)
		log.Printf("Order state cleared for user %s", userID)
	}
}

// GetUserActiveOrder checks if user has an active order (not completed/cancelled)
func (h *Hub) GetUserActiveOrder(userID string) *UserOrderState {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	if h.UserOrders == nil {
		return nil
	}

	state := h.UserOrders[userID]
	if state == nil {
		return nil
	}

	// Check if order is still active (not completed/cancelled)
	if state.Status == "completed" || state.Status == "cancelled" {
		return nil
	}

	return state
}

// BroadcastToUser sends message to all connections of a specific user
func (h *Hub) BroadcastToUser(userID string, msg Message) {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	connections := h.UserConnections[userID]
	for client := range connections {
		select {
		case client.Send <- msg.ToJSON():
		default:
			// Client buffer full, skip
			log.Printf("Failed to send to client %s, buffer full", client.ID)
		}
	}
}

// Close safely closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.closed {
		c.closed = true
		c.Conn.Close()
	}
}