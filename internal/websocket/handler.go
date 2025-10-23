// ABOUTME: HTTP handler for upgrading connections to WebSocket
// ABOUTME: Manages the WebSocket upgrade process and initializes client connections
package websocket

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
	CheckOrigin:       checkOrigin,
}

// checkOrigin validates the Origin header to prevent CSWSH attacks
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}

	// Allow localhost for development (with any port)
	if strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		origin == "http://localhost" ||
		origin == "https://localhost" {
		return true
	}

	// Allow production domain from environment variable
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin != "" && origin == allowedOrigin {
		return true
	}

	// Allow same-origin requests by comparing Origin with Host
	// This handles production deployments without requiring ALLOWED_ORIGIN
	host := r.Host
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	expectedOrigin := scheme + "://" + host
	if origin == expectedOrigin {
		return true
	}

	log.Printf("Rejected WebSocket connection from origin: %s (expected: %s)", origin, expectedOrigin)
	return false
}

// Handler handles WebSocket upgrade requests
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

// ServeHTTP handles the WebSocket connection upgrade
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:                conn,
		send:                make(chan []byte, 256),
		hub:                 h.hub,
		stopInactivityCheck: make(chan struct{}),
	}

	// Don't register yet - wait until we know their sessionID
	// Registration happens in handleCreateSession and handleJoinSession

	// Start client pumps
	go client.writePump()
	go client.readPump()
}
