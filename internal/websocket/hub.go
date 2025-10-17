// ABOUTME: WebSocket hub for managing all client connections and message broadcasting
// ABOUTME: Central coordinator for real-time communication between clients in sessions
package websocket

import (
	"log"
)

// ClientMessage wraps a message with its client
type ClientMessage struct {
	client  *Client
	message *Message
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients (sessionID -> map of clients)
	clients map[string]map[*Client]bool

	// Inbound messages from clients
	process chan *ClientMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Message handler function
	messageHandler func(*Client, *Message)

	// Disconnect handler function
	disconnectHandler func(*Client)
}

// NewHub creates a new Hub
func NewHub(messageHandler func(*Client, *Message)) *Hub {
	return &Hub{
		clients:        make(map[string]map[*Client]bool),
		process:        make(chan *ClientMessage, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		messageHandler: messageHandler,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			sessionClients, exists := h.clients[client.sessionID]
			if !exists {
				sessionClients = make(map[*Client]bool)
				h.clients[client.sessionID] = sessionClients
			}
			sessionClients[client] = true
			log.Printf("Client registered: userId=%s session=%s", client.userID, client.sessionID)

		case client := <-h.unregister:
			if sessionClients, ok := h.clients[client.sessionID]; ok {
				if _, ok := sessionClients[client]; ok {
					delete(sessionClients, client)
					close(client.send)
					log.Printf("Client unregistered: userId=%s session=%s", client.userID, client.sessionID)

					// Call disconnect handler if registered
					if h.disconnectHandler != nil {
						h.disconnectHandler(client)
					}

					// Remove session if no clients left
					if len(sessionClients) == 0 {
						delete(h.clients, client.sessionID)
					}
				}
			}

		case clientMsg := <-h.process:
			// Handle message with the registered handler
			if h.messageHandler != nil {
				h.messageHandler(clientMsg.client, clientMsg.message)
			}
		}
	}
}

// BroadcastToSession sends a message to all clients in a session
func (h *Hub) BroadcastToSession(sessionID string, message *Message) {
	sessionClients, ok := h.clients[sessionID]
	if !ok {
		return
	}

	for client := range sessionClients {
		client.SendMessage(message)
	}
}

// BroadcastToSessionExcept sends a message to all clients except one
func (h *Hub) BroadcastToSessionExcept(sessionID string, exceptUserID string, message *Message) {
	sessionClients, ok := h.clients[sessionID]
	if !ok {
		return
	}

	for client := range sessionClients {
		if client.userID != exceptUserID {
			client.SendMessage(message)
		}
	}
}

// SendToUser sends a message to a specific user in a session
func (h *Hub) SendToUser(sessionID string, userID string, message *Message) {
	sessionClients, ok := h.clients[sessionID]
	if !ok {
		return
	}

	for client := range sessionClients {
		if client.userID == userID {
			client.SendMessage(message)
			return
		}
	}
}

// GetSessionClientCount returns the number of connected clients for a session
func (h *Hub) GetSessionClientCount(sessionID string) int {
	sessionClients, ok := h.clients[sessionID]
	if !ok {
		return 0
	}
	return len(sessionClients)
}

// SetMessageHandler sets the message handler function
func (h *Hub) SetMessageHandler(handler func(*Client, *Message)) {
	h.messageHandler = handler
}

// SetDisconnectHandler sets the disconnect handler function
func (h *Hub) SetDisconnectHandler(handler func(*Client)) {
	h.disconnectHandler = handler
}
