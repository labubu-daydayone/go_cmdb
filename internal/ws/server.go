package ws

import (
	"log"

	socketio "github.com/googollee/go-socket.io"
	"net/http"

	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

var (
	// Server is the global Socket.IO server instance
	Server *socketio.Server
)

// InitServer initializes the Socket.IO server
func InitServer() error {
	// Create server with custom transport options
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now (can be restricted later)
				return true
			},
			},
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool {
					// Allow all origins for now (can be restricted later)
					return true
				},
			},
		},
	})

	// Handle connection event
	server.OnConnect("/", func(s socketio.Conn) error {
		// JWT authentication is handled in auth middleware
		// If we reach here, the connection is authenticated
		log.Printf("[WebSocket] Client connected: %s", s.ID())

		// Send connected confirmation
		s.Emit("connected", map[string]interface{}{
			"ok": true,
		})

		return nil
	})

	// Handle disconnection event
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Printf("[WebSocket] Client disconnected: %s, reason: %s", s.ID(), reason)
	})

	// Handle error event
	server.OnError("/", func(s socketio.Conn, e error) {
		log.Printf("[WebSocket] Error for client %s: %v", s.ID(), e)
	})

	// Register event handlers
	registerEventHandlers(server)

	// Start server goroutine
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("[WebSocket] Server error: %v", err)
		}
	}()

	Server = server
	log.Println("[WebSocket] Socket.IO server initialized")
	return nil
}

// registerEventHandlers registers all Socket.IO event handlers
func registerEventHandlers(server *socketio.Server) {
	// Register request:websites handler
	server.OnEvent("/", "request:websites", handleRequestWebsites)

	log.Println("[WebSocket] Event handlers registered")
}

// BroadcastToRoom broadcasts a message to all clients in a room
func BroadcastToRoom(room string, event string, data interface{}) {
	if Server != nil {
		Server.BroadcastToRoom("/", room, event, data)
	}
}

// BroadcastToAll broadcasts a message to all connected clients
func BroadcastToAll(event string, data interface{}) {
	if Server != nil {
		Server.BroadcastToNamespace("/", event, data)
	}
}
