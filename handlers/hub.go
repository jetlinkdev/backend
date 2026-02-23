package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"

	"jetlink/redis"
)

// Client represents a connected WebSocket client
type Client struct {
	ID        string
	Conn      *websocket.Conn
	Send      chan []byte
	OrderID   *int64  // Current active order ID (stored in Redis, cached here)
	UserID    string  // User identifier
	Role      string  // "customer" | "driver"
	mu        sync.Mutex // Mutex to prevent concurrent close
	closed    bool
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

	// Redis client for order storage
	OrderRedis *redis.OrderRedis
	BidRedis   *redis.BidRedis
}

// NewHub creates a new hub instance
func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

// NewHubWithRedis creates a new hub instance with Redis repositories
func NewHubWithRedis(orderRedis *redis.OrderRedis, bidRedis *redis.BidRedis) *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		OrderRedis: orderRedis,
		BidRedis:   bidRedis,
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

// Close safely closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.closed {
		c.closed = true
		c.Conn.Close()
	}
}