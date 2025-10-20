// ABOUTME: SessionManager handles in-memory storage and retrieval of gratitude circle sessions
// ABOUTME: Provides thread-safe access to session data with lookup by ID or code
package session

import (
	"errors"
	"log"
	"strings"
	"sync"
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
