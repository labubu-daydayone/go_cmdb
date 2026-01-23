package ws

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	socketio "github.com/googollee/go-socket.io"
	"go_cmdb/internal/auth"
)

// AuthMiddleware is a middleware that validates JWT token during WebSocket handshake
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from query parameter or Authorization header
		token := extractToken(r)

		if token == "" {
			log.Printf("[WebSocket] No token provided from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
			return
		}

		// Validate JWT token
		claims, err := auth.ParseToken(token)
		if err != nil {
			log.Printf("[WebSocket] Invalid token from %s: %v", r.RemoteAddr, err)
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Token is valid, add user info to request context
		log.Printf("[WebSocket] Authenticated user: %s (ID: %d)", claims.Username, claims.UID)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// extractToken extracts JWT token from request
// Priority: 1. auth.token query parameter, 2. Authorization header
func extractToken(r *http.Request) string {
	// Try to get token from query parameter (Socket.IO client sends it as auth.token)
	// Socket.IO client: io("url", { auth: { token: "xxx" } })
	// This gets encoded as ?token=xxx in the handshake request
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	// Try to get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	return ""
}

// WrapWithAuth wraps the Socket.IO server with JWT authentication middleware
func WrapWithAuth(server *socketio.Server) http.Handler {
	// Create a custom handler that checks authentication before serving Socket.IO
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check authentication for Socket.IO handshake requests
		// Socket.IO handshake is a GET request to /socket.io/?EIO=4&transport=polling
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/socket.io/") {
			// Extract and validate token
			token := extractToken(r)
			if token == "" {
				log.Printf("[WebSocket] Handshake rejected: No token from %s", r.RemoteAddr)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := auth.ParseToken(token)
			if err != nil {
				log.Printf("[WebSocket] Handshake rejected: Invalid token from %s: %v", r.RemoteAddr, err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			log.Printf("[WebSocket] Handshake accepted: user=%s (ID=%d)", claims.Username, claims.UID)
		}

		// Serve Socket.IO
		server.ServeHTTP(w, r)
	})
}

// GetUserFromConn extracts user information from Socket.IO connection
// Note: This is a placeholder - in production, you should store user info in connection context
func GetUserFromConn(conn socketio.Conn) (userID int64, username string, err error) {
	// In a real implementation, you would store user info during handshake
	// and retrieve it here. For now, we return a placeholder error.
	return 0, "", fmt.Errorf("user info not available in connection context")
}
