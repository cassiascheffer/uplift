// ABOUTME: SessionManager handles in-memory storage and retrieval of gratitude circle sessions
// ABOUTME: Provides thread-safe access to session data with lookup by ID or code
package session

import (
	"errors"
	"log"
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
	m.sessionsByCode[session.Code] = session

	log.Printf("Session created: id=%s code=%s totalSessions=%d", session.ID, session.Code, len(m.sessions))
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

	session, exists := m.sessionsByCode[code]
	if !exists {
		log.Printf("Session lookup failed: code=%s totalSessions=%d", code, len(m.sessions))
		return nil, errors.New("session not found")
	}

	log.Printf("Session found: code=%s id=%s", code, session.ID)
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
	delete(m.sessionsByCode, session.Code)

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
