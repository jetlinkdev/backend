# Jetlink - WebSocket Backend Server

Jetlink is a high-performance WebSocket backend server built with Go. It enables real-time bidirectional communication between clients and the server.

## Features

- Real-time WebSocket communication
- Client management system
- Message broadcasting to all connected clients
- Intent-based message handling for ride-hailing operations
- Structured logging
- Graceful server shutdown
- REST health check endpoints
- Concurrent client handling

## Prerequisites

- Go 1.16 or higher
- Git

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/jetlink.git
cd jetlink
```

2. Install dependencies:
```bash
go mod tidy
```

## Running the Server

### Method 1: Direct execution
```bash
go run cmd/server/main.go
```

### Method 2: Build and run
```bash
go build -o jetlink cmd/server/main.go
./jetlink
```

### Method 3: Hot reload with Air (recommended for development)
First, install Air:
```bash
go install github.com/air-verse/air@latest
```

Then run the server with hot reload:
```bash
air
```

Air will automatically detect file changes and restart the server. The configuration is stored in `.air.toml`.

By default, the server will start on `:8080`. You can specify a different port:
```bash
go run cmd/server/main.go -addr=:9000
```

Or with Air:
```bash
air -build.cmd "go build -o ./tmp/main cmd/server/main.go" -build.bin "./tmp/main -addr=:9000"
```

## Endpoints

- `GET /ws` - WebSocket endpoint for real-time communication
- `GET /health` - Health check endpoint
- `GET /clients` - Returns the number of currently connected clients

## Message Format

The Jetlink server uses a JSON-based message format with the following structure:

```json
{
  "intent": "intent_name",
  "data": {
    // Intent-specific data
  },
  "timestamp": 1234567890,
  "clientId": "unique_client_id"
}
```

## Supported Intents

### create_order
Used to create a new ride-hailing order.

Example request:
```json
{
  "intent": "create_order",
  "data": {
    "pickup": "Stasiun Jember",
    "destination": "Alun-alun Jember",
    "notes": "",
    "time": "Segera",
    "payment": "Cash",
    "userId": "user123"
  }
}
```

Response (sent back to the requesting client):
```json
{
  "intent": "order_created",
  "data": {
    "id": "order_1234567890_clientid",
    "userId": "user123",
    "pickup": "Stasiun Jember",
    "destination": "Alun-alun Jember",
    "notes": "",
    "time": "Segera",
    "payment": "Cash",
    "status": "pending",
    "fare": 15000,
    "createdAt": 1234567890,
    "updatedAt": 1234567890
  },
  "timestamp": 1234567890,
  "clientId": "clientid"
}
```

Broadcast to other clients (drivers):
```json
{
  "intent": "new_order_available",
  "data": {
    "id": "order_1234567890_clientid",
    "userId": "user123",
    "pickup": "Stasiun Jember",
    "destination": "Alun-alun Jember",
    "notes": "",
    "time": "Segera",
    "payment": "Cash",
    "status": "pending",
    "fare": 15000,
    "createdAt": 1234567890,
    "updatedAt": 1234567890
  },
  "timestamp": 1234567890,
  "clientId": "clientid"
}
```

### ping
Used to check connectivity.

Request:
```json
{
  "intent": "ping",
  "data": {}
}
```

Response:
```json
{
  "intent": "pong",
  "data": {},
  "timestamp": 1234567890,
  "clientId": "clientid"
}
```

## Testing the WebSocket Connection

You can test the WebSocket connection using a browser console:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function() {
  console.log('Connected to Jetlink server');
};

ws.onmessage = function(event) {
  console.log('Received:', event.data);
};

ws.onclose = function() {
  console.log('Disconnected from Jetlink server');
};

// Send a create_order message
const createOrderMessage = {
  "intent": "create_order",
  "data": {
    "pickup": "Stasiun Jember",
    "destination": "Alun-alun Jember",
    "notes": "",
    "time": "Segera",
    "payment": "Cash",
    "userId": "user123"
  }
};
ws.send(JSON.stringify(createOrderMessage));
```

Or use a WebSocket testing tool like [wscat](https://www.npmjs.com/package/wscat):
```bash
npm install -g wscat
wscat -c ws://localhost:8080/ws
```

## Architecture

- `main.go` - Entry point of the application, sets up the HTTP server and WebSocket routes
- `handlers/hub.go` - Manages WebSocket connections, client registration, and message broadcasting
- `utils/logger.go` - Provides structured logging functionality
- `models/` - Contains data models (to be added as needed)

## Dependencies

- `github.com/gorilla/websocket` - WebSocket implementation
- `github.com/gorilla/mux` - HTTP request router
- `github.com/gorilla/handlers` - HTTP middleware (CORS)

## Development Tools

- `github.com/air-verse/air` - Live reload utility for Go applications (optional, for development)

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.