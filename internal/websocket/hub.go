// ABOUTME: WebSocket hub for managing all client connections and message broadcasting
// ABOUTME: Central coordinator for real-time communication between clients in sessions
package websocket

import (
	"log"
	"sync"
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

	// Mutex to protect clients map
	clientsMu sync.RWMutex

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
			h.clientsMu.Lock()
			sessionClients, exists := h.clients[client.sessionID]
			if !exists {
				sessionClients = make(map[*Client]bool)
				h.clients[client.sessionID] = sessionClients
			}
			sessionClients[client] = true
			h.clientsMu.Unlock()
			log.Printf("Client registered: userId=%s session=%s", client.userID, client.sessionID)

		case client := <-h.unregister:
			h.clientsMu.Lock()
			if sessionClients, ok := h.clients[client.sessionID]; ok {
				if _, ok := sessionClients[client]; ok {
					delete(sessionClients, client)
					client.closeSendChannel()
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
			h.clientsMu.Unlock()

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
	h.clientsMu.RLock()
	sessionClients, ok := h.clients[sessionID]
	if !ok {
		h.clientsMu.RUnlock()
		return
	}

	// Copy client pointers to avoid holding lock during send
	clients := make([]*Client, 0, len(sessionClients))
	for client := range sessionClients {
		clients = append(clients, client)
	}
	h.clientsMu.RUnlock()

	for _, client := range clients {
		client.SendMessage(message)
	}
}

// BroadcastToSessionExcept sends a message to all clients except one
func (h *Hub) BroadcastToSessionExcept(sessionID string, exceptUserID string, message *Message) {
	h.clientsMu.RLock()
	sessionClients, ok := h.clients[sessionID]
	if !ok {
		h.clientsMu.RUnlock()
		return
	}

	// Copy client pointers to avoid holding lock during send
	clients := make([]*Client, 0, len(sessionClients))
	for client := range sessionClients {
		if client.userID != exceptUserID {
			clients = append(clients, client)
		}
	}
	h.clientsMu.RUnlock()

	for _, client := range clients {
		client.SendMessage(message)
	}
}

// SendToUser sends a message to a specific user in a session
func (h *Hub) SendToUser(sessionID string, userID string, message *Message) {
	h.clientsMu.RLock()
	sessionClients, ok := h.clients[sessionID]
	if !ok {
		h.clientsMu.RUnlock()
		return
	}

	var targetClient *Client
	for client := range sessionClients {
		if client.userID == userID {
			targetClient = client
			break
		}
	}
	h.clientsMu.RUnlock()

	if targetClient != nil {
		targetClient.SendMessage(message)
	}
}

// GetSessionClientCount returns the number of connected clients for a session
func (h *Hub) GetSessionClientCount(sessionID string) int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

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
