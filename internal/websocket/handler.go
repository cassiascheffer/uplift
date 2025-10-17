// ABOUTME: HTTP handler for upgrading connections to WebSocket
// ABOUTME: Manages the WebSocket upgrade process and initializes client connections
package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, implement proper origin checking
		return true
	},
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
		conn: conn,
		send: make(chan []byte, 256),
		hub:  h.hub,
	}

	// Don't register yet - wait until we know their sessionID
	// Registration happens in handleCreateSession and handleJoinSession

	// Start client pumps
	go client.writePump()
	go client.readPump()
}
