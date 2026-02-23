package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	hubhandlers "jetlink/handlers"
	"jetlink/database"
	"jetlink/intents"
	"jetlink/utils"
	"jetlink/constants"
)

var (
	Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from any origin
			// In production, you should validate origins properly
			return true
		},
	}

	// WebSocket connection settings
	writeWait      = 10 * time.Second  // Timeout for individual write operations
	pongWait       = 60 * time.Second  // Time to wait for pong before considering connection dead
	pingPeriod     = (pongWait * 9) / 10 // Send ping at 90% of pongWait
	maxMessageSize = int64(4096)       // Increased from 512 to handle larger JSON payloads
)

// clientCounter for generating unique client IDs
var clientCounter uint64

// ConnectionsHandler handles WebSocket connections
func ConnectionsHandler(w http.ResponseWriter, r *http.Request, hub *hubhandlers.Hub, logger *utils.Logger, repo *database.OrderRepository) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Generate unique client ID using UUID + counter for stability
	clientID := fmt.Sprintf("client-%s-%d", uuid.New().String()[:8], atomic.AddUint64(&clientCounter, 1))

	client := &hubhandlers.Client{
		ID:   clientID,
		Conn: conn,
		Send: make(chan []byte, 512), // Buffer for outgoing messages
	}

	// Set initial ReadDeadline
	conn.SetReadDeadline(time.Now().Add(pongWait))

	// Set PongHandler to automatically refresh ReadDeadline on pong received
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Set maximum message size
	conn.SetReadLimit(maxMessageSize)

	hub.Register <- client

	// Start sending messages to the client
	go sendMessages(client, logger)

	// Start listening for messages from the client
	listenForMessages(client, hub, logger, repo)
}

// sendMessages handles sending messages to the client with proper deadline management
func sendMessages(client *hubhandlers.Client, logger *utils.Logger) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			// Set write deadline before writing
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// Hub closed the channel - send proper close frame
				client.Conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server closing"),
					time.Now().Add(writeWait),
				)
				client.Close()
				return
			}

			// Send message to client
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Info(fmt.Sprintf("Failed to send message to client %s: %v", client.ID, err))
				client.Close()
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive and detect zombie connections
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Info(fmt.Sprintf("Ping failed for client %s, closing connection: %v", client.ID, err))
				client.Close()
				return
			}
		}
	}
}

// listenForMessages handles incoming messages from the client with proper deadline management
func listenForMessages(client *hubhandlers.Client, hub *hubhandlers.Hub, logger *utils.Logger, repo *database.OrderRepository) {
	defer func() {
		// Ensure client is unregistered and connection is closed
		hub.Unregister <- client
		client.Close()
	}()

	// ReadDeadline is managed by SetReadDeadline and PongHandler
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			// Check for specific close codes
			if closeErr, ok := err.(*websocket.CloseError); ok {
				// Normal close codes - log as info
				if closeErr.Code == websocket.CloseNormalClosure ||
					closeErr.Code == websocket.CloseGoingAway ||
					closeErr.Code == 1005 { // No Status Received
					logger.Info(fmt.Sprintf("Client %s disconnected normally (code: %d)", client.ID, closeErr.Code))
					break
				}
			}

			// Check for timeout error (connection zombie)
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				logger.Info(fmt.Sprintf("Client %s connection timed out (zombie detection)", client.ID))
				break
			}

			// Log other unexpected errors
			if websocket.IsUnexpectedCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				logger.Error(fmt.Sprintf("WebSocket error for client %s: %v", client.ID, err))
			} else {
				logger.Info(fmt.Sprintf("Client %s disconnected: %v", client.ID, err))
			}
			break
		}

		// Refresh ReadDeadline after successfully receiving a message
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))

		// Parse the incoming message to determine intent
		var incomingMsg hubhandlers.Message
		if err := json.Unmarshal(message, &incomingMsg); err != nil {
			logger.Error(fmt.Sprintf("Failed to parse message from client %s: %v", client.ID, err))

			// Send error response back to client (non-blocking)
			errorMsg := hubhandlers.Message{
				Intent:    constants.IntentError,
				Data:      map[string]string{"message": "Invalid message format"},
				Timestamp: time.Now().Unix(),
				ClientID:  client.ID,
			}
			select {
			case client.Send <- errorMsg.ToJSON():
			default:
				logger.Info(fmt.Sprintf("Client %s send buffer full, dropping error message", client.ID))
			}
			continue
		}

		// Log received message
		logger.Info(fmt.Sprintf("Received message from client %s with intent: %s", client.ID, incomingMsg.Intent))

		// Handle different intents
		switch incomingMsg.Intent {
		case constants.IntentAuth:
			intents.HandleAuth(client, hub, logger, incomingMsg, repo)
		case constants.IntentDriverRegistration:
			intents.HandleDriverRegistration(client, hub, logger, incomingMsg, repo)
		case constants.IntentCheckDriverStatus:
			intents.HandleCheckDriverStatus(client, hub, logger, incomingMsg, repo)
		case constants.IntentCreateOrder:
			intents.HandleCreateOrder(client, hub, logger, incomingMsg, repo)
		case constants.IntentCancelOrder:
			intents.HandleCancelOrder(client, hub, logger, incomingMsg, repo)
		case constants.IntentSubmitBid:
			intents.HandleSubmitBid(client, hub, logger, incomingMsg, repo)
		case constants.IntentSelectBid:
			intents.HandleSelectBid(client, hub, logger, incomingMsg, repo)
		case constants.IntentDriverArrived:
			intents.HandleDriverArrived(client, hub, logger, incomingMsg, repo)
		case constants.IntentCompleteTrip:
			intents.HandleCompleteTrip(client, hub, logger, incomingMsg, repo)
		case constants.IntentPing:
			intents.HandlePing(client)
		default:
			// Broadcast other messages to all other clients
			incomingMsg.ClientID = client.ID
			incomingMsg.Timestamp = time.Now().Unix()
			hub.BroadcastMessage(incomingMsg)
		}
	}
}
