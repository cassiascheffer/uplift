package session

import (
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	hostName := "Test Host"
	sess := NewSession(hostName)

	if sess.ID == "" {
		t.Error("Expected session ID to be generated")
	}

	if sess.Code == "" {
		t.Error("Expected session code to be generated")
	}

	if len(sess.Code) > 6 {
		t.Errorf("Session code too long: got %d characters, max 6", len(sess.Code))
	}

	if sess.Phase != PhaseJoining {
		t.Errorf("Expected initial phase to be JOINING, got %s", sess.Phase)
	}

	if len(sess.Participants) != 1 {
		t.Errorf("Expected 1 participant (host), got %d", len(sess.Participants))
	}

	if sess.HostID == "" {
		t.Error("Expected host ID to be set")
	}

	// Verify host participant exists
	host, exists := sess.Participants[sess.HostID]
	if !exists {
		t.Error("Host participant not found in session")
	}

	if !host.IsHost {
		t.Error("Expected host participant to have IsHost=true")
	}

	if host.Name != hostName {
		t.Errorf("Expected host name %s, got %s", hostName, host.Name)
	}

	if sess.CurrentTurn != 0 {
		t.Errorf("Expected current turn to be 0, got %d", sess.CurrentTurn)
	}

	if len(sess.Notes) != 0 {
		t.Errorf("Expected 0 notes initially, got %d", len(sess.Notes))
	}
}

func TestAddParticipant(t *testing.T) {
	sess := NewSession("Host")

	// Add a participant
	participant, err := sess.AddParticipant("Alice")
	if err != nil {
		t.Fatalf("Failed to add participant: %v", err)
	}

	if participant.Name != "Alice" {
		t.Errorf("Expected participant name Alice, got %s", participant.Name)
	}

	if participant.IsHost {
		t.Error("Expected non-host participant to have IsHost=false")
	}

	if len(sess.Participants) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(sess.Participants))
	}

	// Try to add participant after writing phase started
	sess.Phase = PhaseWriting
	_, err = sess.AddParticipant("Bob")
	if err == nil {
		t.Error("Expected error when adding participant after session started")
	}
}

func TestTransitionToWriting(t *testing.T) {
	sess := NewSession("Host")
	sess.AddParticipant("Alice")

	err := sess.TransitionToWriting()
	if err != nil {
		t.Fatalf("Failed to transition to writing: %v", err)
	}

	if sess.Phase != PhaseWriting {
		t.Errorf("Expected phase to be WRITING, got %s", sess.Phase)
	}
}

func TestAddNote(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")
	sess.TransitionToWriting()

	// Add a note
	err := sess.AddNote(sess.HostID, alice.ID, "Great work!")
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	if len(sess.Notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(sess.Notes))
	}

	note := sess.Notes[0]
	if note.AuthorID != sess.HostID {
		t.Errorf("Expected author to be host, got %s", note.AuthorID)
	}

	if note.RecipientID != alice.ID {
		t.Errorf("Expected recipient to be Alice, got %s", note.RecipientID)
	}

	if note.Content != "Great work!" {
		t.Errorf("Expected content 'Great work!', got %s", note.Content)
	}

	if note.Read {
		t.Error("Expected new note to be unread")
	}
}

func TestTransitionToReading(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")
	sess.TransitionToWriting()

	// Add required notes (each person writes to each other person)
	sess.AddNote(sess.HostID, alice.ID, "Note 1")
	sess.AddNote(alice.ID, sess.HostID, "Note 2")

	err := sess.TransitionToReading()
	if err != nil {
		t.Fatalf("Failed to transition to reading: %v", err)
	}

	if sess.Phase != PhaseReading {
		t.Errorf("Expected phase to be READING, got %s", sess.Phase)
	}

	currentReader := sess.GetCurrentReader()
	if currentReader == nil {
		t.Error("Expected current reader to be set")
	}
}

func TestMarkNoteAsRead(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")
	sess.TransitionToWriting()
	sess.AddNote(sess.HostID, alice.ID, "Great work!")

	note := sess.Notes[0]
	err := sess.MarkNoteAsRead(note.ID)
	if err != nil {
		t.Fatalf("Failed to mark note as read: %v", err)
	}

	if !sess.Notes[0].Read {
		t.Error("Expected note to be marked as read")
	}
}

func TestAdvanceTurn(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")
	bob, _ := sess.AddParticipant("Bob")
	sess.TransitionToWriting()

	// Add all required notes
	sess.AddNote(sess.HostID, alice.ID, "Note 1")
	sess.AddNote(sess.HostID, bob.ID, "Note 2")
	sess.AddNote(alice.ID, sess.HostID, "Note 3")
	sess.AddNote(alice.ID, bob.ID, "Note 4")
	sess.AddNote(bob.ID, sess.HostID, "Note 5")
	sess.AddNote(bob.ID, alice.ID, "Note 6")

	sess.TransitionToReading()

	initialReader := sess.GetCurrentReader()
	sess.AdvanceTurn()
	nextReader := sess.GetCurrentReader()

	if initialReader.ID == nextReader.ID {
		t.Error("Expected reader to change after advancing turn")
	}
}

func TestRemoveParticipant(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")

	removed, err := sess.RemoveParticipant(alice.ID)
	if err != nil {
		t.Fatalf("Failed to remove participant: %v", err)
	}

	if removed.ID != alice.ID {
		t.Errorf("Expected removed participant ID %s, got %s", alice.ID, removed.ID)
	}

	if len(sess.Participants) != 1 {
		t.Errorf("Expected 1 participant remaining, got %d", len(sess.Participants))
	}

	// Try to remove non-existent participant
	_, err = sess.RemoveParticipant("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent participant")
	}
}

func TestGetAvailableNotesForReader(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")
	bob, _ := sess.AddParticipant("Bob")
	sess.TransitionToWriting()

	// Add notes
	sess.AddNote(sess.HostID, alice.ID, "Host to Alice")
	sess.AddNote(alice.ID, bob.ID, "Alice to Bob")
	sess.AddNote(bob.ID, sess.HostID, "Bob to Host")

	sess.TransitionToReading()

	// Host should not see notes they wrote or received
	availableForHost := sess.GetAvailableNotesForReader(sess.HostID)

	// Host should only see "Alice to Bob" note (not authored by or for host)
	if len(availableForHost) != 1 {
		t.Errorf("Expected 1 available note for host, got %d", len(availableForHost))
	}
}

func TestSessionCompletion(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")
	sess.TransitionToWriting()

	// Add notes
	sess.AddNote(sess.HostID, alice.ID, "Note 1")
	sess.AddNote(alice.ID, sess.HostID, "Note 2")

	sess.TransitionToReading()

	// Mark all notes as read
	for _, note := range sess.Notes {
		sess.MarkNoteAsRead(note.ID)
	}

	// Advance turn should complete the session
	sess.AdvanceTurn()

	if sess.Phase != PhaseComplete {
		t.Errorf("Expected phase to be COMPLETE, got %s", sess.Phase)
	}

	if sess.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}

	if time.Since(*sess.CompletedAt) > time.Second {
		t.Error("CompletedAt timestamp should be recent")
	}
}

func TestGetParticipantList(t *testing.T) {
	sess := NewSession("Host")
	alice, _ := sess.AddParticipant("Alice")
	bob, _ := sess.AddParticipant("Bob")

	participants := sess.GetParticipantList()

	if len(participants) != 3 {
		t.Fatalf("Expected 3 participants, got %d", len(participants))
	}

	// Verify all participants are in the list
	foundHost := false
	foundAlice := false
	foundBob := false

	for _, p := range participants {
		if p.ID == sess.HostID {
			foundHost = true
		}
		if p.ID == alice.ID {
			foundAlice = true
		}
		if p.ID == bob.ID {
			foundBob = true
		}
	}

	if !foundHost || !foundAlice || !foundBob {
		t.Error("Expected all participants to be in the list")
	}
}

func TestIDGeneration(t *testing.T) {
	// Test that IDs are unique
	id1 := generateID()
	id2 := generateID()

	if id1 == id2 {
		t.Error("Expected unique IDs")
	}

	if id1 == "" || id2 == "" {
		t.Error("Expected non-empty IDs")
	}
}

func TestSessionCodeGeneration(t *testing.T) {
	// Test that session codes are unique
	code1 := generateSessionCode()
	code2 := generateSessionCode()

	if code1 == code2 {
		t.Error("Expected unique session codes (may rarely fail due to randomness)")
	}

	if len(code1) > 6 || len(code2) > 6 {
		t.Error("Session codes should be maximum 6 characters")
	}

	if code1 == "" || code2 == "" {
		t.Error("Expected non-empty session codes")
	}
}
