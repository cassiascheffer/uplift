package session

import (
	"context"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}

	if manager.sessionsByCode == nil {
		t.Error("Expected sessionsByCode map to be initialized")
	}

	count := manager.GetActiveSessionCount()
	if count != 0 {
		t.Errorf("Expected 0 active sessions, got %d", count)
	}
}

func TestCreateSession(t *testing.T) {
	manager := NewManager()

	sess := manager.CreateSession("Test Host")

	if sess == nil {
		t.Fatal("Expected session to be created")
	}

	if sess.ID == "" {
		t.Error("Expected session to have an ID")
	}

	if sess.Code == "" {
		t.Error("Expected session to have a code")
	}

	count := manager.GetActiveSessionCount()
	if count != 1 {
		t.Errorf("Expected 1 active session, got %d", count)
	}
}

func TestGetSessionByID(t *testing.T) {
	manager := NewManager()
	createdSession := manager.CreateSession("Host")

	// Get existing session
	sess, err := manager.GetSessionByID(createdSession.ID)
	if err != nil {
		t.Fatalf("Failed to get session by ID: %v", err)
	}

	if sess.ID != createdSession.ID {
		t.Errorf("Expected session ID %s, got %s", createdSession.ID, sess.ID)
	}

	// Try to get non-existent session
	_, err = manager.GetSessionByID("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent session")
	}
}

func TestGetSessionByCode(t *testing.T) {
	manager := NewManager()
	createdSession := manager.CreateSession("Host")

	// Get existing session (case-insensitive)
	sess, err := manager.GetSessionByCode(createdSession.Code)
	if err != nil {
		t.Fatalf("Failed to get session by code: %v", err)
	}

	if sess.ID != createdSession.ID {
		t.Errorf("Expected session ID %s, got %s", createdSession.ID, sess.ID)
	}

	// Test case-insensitive lookup
	lowerCode := "abc123"
	upperCode := "ABC123"
	manager2 := NewManager()
	testSession := manager2.CreateSession("Test")
	testSession.Code = lowerCode
	manager2.sessionsByCode[upperCode] = testSession

	retrieved, err := manager2.GetSessionByCode(lowerCode)
	if err != nil {
		t.Fatalf("Case-insensitive lookup failed: %v", err)
	}

	if retrieved.ID != testSession.ID {
		t.Error("Expected case-insensitive code lookup to work")
	}

	// Try to get non-existent session
	_, err = manager.GetSessionByCode("NONEXISTENT")
	if err == nil {
		t.Error("Expected error when getting non-existent session")
	}
}

func TestRemoveSession(t *testing.T) {
	manager := NewManager()
	sess := manager.CreateSession("Host")

	err := manager.RemoveSession(sess.ID)
	if err != nil {
		t.Fatalf("Failed to remove session: %v", err)
	}

	count := manager.GetActiveSessionCount()
	if count != 0 {
		t.Errorf("Expected 0 active sessions after removal, got %d", count)
	}

	// Verify session is also removed from sessionsByCode
	_, err = manager.GetSessionByCode(sess.Code)
	if err == nil {
		t.Error("Expected session to be removed from sessionsByCode map")
	}

	// Try to remove non-existent session
	err = manager.RemoveSession("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent session")
	}
}

func TestGetAllSessions(t *testing.T) {
	manager := NewManager()

	manager.CreateSession("Host 1")
	manager.CreateSession("Host 2")
	manager.CreateSession("Host 3")

	sessions := manager.GetAllSessions()

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}

func TestConcurrentSessionAccess(t *testing.T) {
	manager := NewManager()

	// Create sessions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			manager.CreateSession("Concurrent Host")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	count := manager.GetActiveSessionCount()
	if count != 10 {
		t.Errorf("Expected 10 sessions after concurrent creation, got %d", count)
	}
}

func TestCleanupCompletedSessions(t *testing.T) {
	manager := NewManager()

	// Create a completed session older than 1 hour
	oldSession := manager.CreateSession("Old Host")
	oldTime := time.Now().Add(-2 * time.Hour)
	oldSession.Phase = PhaseComplete
	oldSession.CompletedAt = &oldTime

	// Create a recent completed session
	recentSession := manager.CreateSession("Recent Host")
	recentTime := time.Now().Add(-30 * time.Minute)
	recentSession.Phase = PhaseComplete
	recentSession.CompletedAt = &recentTime

	// Create an active session
	activeSession := manager.CreateSession("Active Host")

	// Run cleanup
	manager.cleanupSessions()

	// Old completed session should be removed
	_, err := manager.GetSessionByID(oldSession.ID)
	if err == nil {
		t.Error("Expected old completed session to be removed")
	}

	// Recent completed session should remain
	_, err = manager.GetSessionByID(recentSession.ID)
	if err != nil {
		t.Error("Expected recent completed session to remain")
	}

	// Active session should remain
	_, err = manager.GetSessionByID(activeSession.ID)
	if err != nil {
		t.Error("Expected active session to remain")
	}
}

func TestCleanupAbandonedSessions(t *testing.T) {
	manager := NewManager()

	// Create session and remove all participants
	abandonedSession := manager.CreateSession("Host")
	for id := range abandonedSession.Participants {
		abandonedSession.RemoveParticipant(id)
	}

	// Create normal session
	normalSession := manager.CreateSession("Normal Host")

	// Run cleanup
	manager.cleanupSessions()

	// Abandoned session should be removed
	_, err := manager.GetSessionByID(abandonedSession.ID)
	if err == nil {
		t.Error("Expected abandoned session to be removed")
	}

	// Normal session should remain
	_, err = manager.GetSessionByID(normalSession.ID)
	if err != nil {
		t.Error("Expected normal session to remain")
	}
}

func TestStartCleanupRoutine(t *testing.T) {
	manager := NewManager()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start cleanup routine
	done := make(chan bool)
	go func() {
		manager.StartCleanupRoutine(ctx)
		done <- true
	}()

	// Wait for context cancellation
	select {
	case <-done:
		// Cleanup routine exited as expected
	case <-time.After(200 * time.Millisecond):
		t.Error("Cleanup routine did not exit after context cancellation")
	}
}

func TestMultipleSessionsByDifferentHosts(t *testing.T) {
	manager := NewManager()

	session1 := manager.CreateSession("Alice")
	session2 := manager.CreateSession("Bob")
	session3 := manager.CreateSession("Charlie")

	if session1.Code == session2.Code || session2.Code == session3.Code || session1.Code == session3.Code {
		t.Error("Expected unique session codes (may rarely fail due to randomness)")
	}

	// Verify all can be retrieved
	_, err1 := manager.GetSessionByCode(session1.Code)
	_, err2 := manager.GetSessionByCode(session2.Code)
	_, err3 := manager.GetSessionByCode(session3.Code)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Error("Failed to retrieve all sessions by code")
	}
}
