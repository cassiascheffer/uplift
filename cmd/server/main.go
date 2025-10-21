// ABOUTME: Entry point for the uplift web server
// ABOUTME: Sets up HTTP routes and WebSocket handling for real-time session synchronization
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cassiascheffer/uplift/internal/session"
	"github.com/cassiascheffer/uplift/internal/websocket"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create context that will be cancelled on SIGINT/SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create session manager
	sessionManager := session.NewManager()

	// Start session cleanup routine in background with cancellable context
	go sessionManager.StartCleanupRoutine(ctx)

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

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: nil, // Use DefaultServeMux
	}

	// Start server in background
	go func() {
		log.Printf("Starting uplift server on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Printf("Shutdown signal received, starting graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Printf("Server shutdown complete")
	}
}
