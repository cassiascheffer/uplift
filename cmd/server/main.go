// ABOUTME: Entry point for the uplift web server
// ABOUTME: Sets up HTTP routes and WebSocket handling for real-time session synchronization
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/cassiascheffer/uplift/internal/session"
	"github.com/cassiascheffer/uplift/internal/websocket"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create session manager
	sessionManager := session.NewManager()

	// Start session cleanup routine in background
	go sessionManager.StartCleanupRoutine(context.Background())

	// Create WebSocket hub
	hub := websocket.NewHub(nil)

	// Create message handler
	messageHandler := websocket.NewMessageHandler(hub, sessionManager)

	// Set the message handler on the hub
	hub.SetMessageHandler(messageHandler.HandleMessage)

	// Set the disconnect handler on the hub
	hub.SetDisconnectHandler(messageHandler.HandleClientDisconnect)

	// Start hub in background
	go hub.Run()

	// Create WebSocket handler
	wsHandler := websocket.NewHandler(hub)

	// Register routes
	http.Handle("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("Starting uplift server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
