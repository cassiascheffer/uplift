// ABOUTME: SessionManager handles in-memory storage and retrieval of gratitude circle sessions
// ABOUTME: Provides thread-safe access to session data with lookup by ID or code
package session

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"
)

// Manager manages all active sessions in memory
type Manager struct {
	sessions       map[string]*Session // sessionID -> Session
	sessionsByCode map[string]*Session // sessionCode -> Session
	mu             sync.RWMutex
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions:       make(map[string]*Session),
		sessionsByCode: make(map[string]*Session),
	}
}

// CreateSession creates a new session and stores it
func (m *Manager) CreateSession(hostName string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := NewSession(hostName)
	m.sessions[session.ID] = session
	// Normalize session code to uppercase for consistent lookups
	normalizedCode := strings.ToUpper(strings.TrimSpace(session.Code))
	m.sessionsByCode[normalizedCode] = session

	log.Printf("Session created: id=%s code=%s totalSessions=%d", session.ID, normalizedCode, len(m.sessions))
	return session
}

// GetSessionByID retrieves a session by its ID
func (m *Manager) GetSessionByID(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, errors.New("session not found")
	}

	return session, nil
}

// GetSessionByCode retrieves a session by its code (case-insensitive)
func (m *Manager) GetSessionByCode(code string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Normalize code to uppercase for case-insensitive lookup
	normalizedCode := strings.ToUpper(strings.TrimSpace(code))

	session, exists := m.sessionsByCode[normalizedCode]
	if !exists {
		log.Printf("Session lookup failed: code=%s (normalized=%s) totalSessions=%d", code, normalizedCode, len(m.sessions))
		return nil, errors.New("session not found")
	}

	log.Printf("Session found: code=%s id=%s", normalizedCode, session.ID)
	return session, nil
}

// RemoveSession removes a session from the manager
func (m *Manager) RemoveSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return errors.New("session not found")
	}

	delete(m.sessions, sessionID)
	// Normalize session code for deletion
	normalizedCode := strings.ToUpper(strings.TrimSpace(session.Code))
	delete(m.sessionsByCode, normalizedCode)

	return nil
}

// GetActiveSessionCount returns the number of active sessions
func (m *Manager) GetActiveSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// GetAllSessions returns all active sessions (for debugging/admin purposes)
func (m *Manager) GetAllSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// StartCleanupRoutine starts a background goroutine that periodically cleans up old sessions
func (m *Manager) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Printf("Session cleanup routine started (runs every 5 minutes)")

	for {
		select {
		case <-ctx.Done():
			log.Printf("Session cleanup routine stopped")
			return
		case <-ticker.C:
			m.cleanupSessions()
		}
	}
}

// cleanupSessions removes old completed sessions and abandoned sessions
func (m *Manager) cleanupSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	completedThreshold := now.Add(-1 * time.Hour)
	cleanedCount := 0

	for sessionID, session := range m.sessions {
		session.mu.RLock()
		shouldRemove := false
		reason := ""

		// Remove abandoned sessions (no participants)
		if len(session.Participants) == 0 {
			shouldRemove = true
			reason = "abandoned (no participants)"
		} else if session.Phase == PhaseComplete && session.CompletedAt != nil {
			// Remove completed sessions older than 1 hour
			if session.CompletedAt.Before(completedThreshold) {
				shouldRemove = true
				reason = "completed over 1 hour ago"
			}
		}

		sessionCode := session.Code
		session.mu.RUnlock()

		if shouldRemove {
			delete(m.sessions, sessionID)
			normalizedCode := strings.ToUpper(strings.TrimSpace(sessionCode))
			delete(m.sessionsByCode, normalizedCode)
			cleanedCount++
			log.Printf("Cleaned up session: id=%s code=%s reason=%s", sessionID, sessionCode, reason)
		}
	}

	if cleanedCount > 0 {
		log.Printf("Session cleanup complete: removed=%d remaining=%d", cleanedCount, len(m.sessions))
	}
}
