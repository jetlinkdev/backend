package intents

import (
	"time"

	hubhandlers "jetlink/handlers"
	"jetlink/constants"
)

// HandlePing handles the ping intent
func HandlePing(client *hubhandlers.Client) {
	// Respond with pong
	pongMsg := hubhandlers.Message{
		Intent:    constants.IntentPong,
		Data:      map[string]interface{}{},
		Timestamp: time.Now().Unix(),
		ClientID:  client.ID,
	}
	client.Send <- pongMsg.ToJSON()
}