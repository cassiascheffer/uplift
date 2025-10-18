// ABOUTME: WebSocket client connection management for real-time session synchronization
// ABOUTME: Handles individual client connections with read/write pumps and message routing
package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum inactivity time before disconnecting (30 minutes)
	inactivityTimeout = 30 * time.Minute

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client represents a WebSocket client connection
type Client struct {
	// The WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// The hub managing this client
	hub *Hub

	// Session ID this client is connected to
	sessionID string

	// User ID for this client
	userID string

	// User name for this client
	userName string

	// Last activity timestamp for inactivity timeout
	lastActivity time.Time
}

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	SessionID string                 `json:"sessionId,omitempty"`
	UserID    string                 `json:"userId,omitempty"`
	UserName  string                 `json:"userName,omitempty"`
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.lastActivity = time.Now()
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastActivity = time.Now()
		return nil
	})

	// Start goroutine to check for inactivity
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if time.Since(c.lastActivity) > inactivityTimeout {
				log.Printf("Client inactive for %v, disconnecting: userId=%s session=%s", inactivityTimeout, c.userID, c.sessionID)
				// Send timeout message before closing
				timeoutMsg := &Message{
					Type: "timeout",
					Data: map[string]interface{}{
						"message": "Disconnected due to inactivity. Please start again.",
					},
				}
				c.SendMessage(timeoutMsg)
				time.Sleep(100 * time.Millisecond) // Give time for message to send
				// Close with policy violation code (1008) for timeout
				c.conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(1008, "Inactivity timeout"),
					time.Now().Add(writeWait),
				)
				c.conn.Close()
				return
			}
		}
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		// Update last activity timestamp
		c.lastActivity = time.Now()

		// Parse message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error parsing message: %v", err)
			continue
		}

		// Set client context on message
		msg.SessionID = c.sessionID
		msg.UserID = c.userID
		msg.UserName = c.userName

		// Send to hub for processing
		c.hub.process <- &ClientMessage{
			client:  c,
			message: &msg,
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to this client
func (c *Client) SendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		// Client's send buffer is full, close connection
		close(c.send)
		return nil
	}
}
