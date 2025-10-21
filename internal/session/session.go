// ABOUTME: Core session data structures and business logic for gratitude circles
// ABOUTME: Handles session lifecycle, participants, notes, and phase transitions
package session

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// Phase represents the current phase of a gratitude circle session
type Phase string

const (
	PhaseJoining  Phase = "JOINING"
	PhaseWriting  Phase = "WRITING"
	PhaseReading  Phase = "READING"
	PhaseComplete Phase = "COMPLETE"
)

// Participant represents a person in the session
type Participant struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	IsHost bool      `json:"isHost"`
	JoinedAt time.Time `json:"joinedAt"`
}

// Note represents a gratitude note
type Note struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	AuthorID    string `json:"authorId"`
	RecipientID string `json:"recipientId"`
	Read        bool   `json:"read"`
}

// Session represents a gratitude circle session
type Session struct {
	ID           string                  `json:"id"`
	Code         string                  `json:"code"`
	Phase        Phase                   `json:"phase"`
	Participants map[string]*Participant `json:"participants"`
	Notes        []*Note                 `json:"notes"`
	CreatedAt    time.Time               `json:"createdAt"`
	HostID       string                  `json:"hostId"`
	CurrentTurn  int                     `json:"currentTurn"` // Index of current reader
	mu           sync.RWMutex
}

// NewSession creates a new session with a unique code
func NewSession(hostName string) *Session {
	code := generateSessionCode()
	hostID := generateID()

	host := &Participant{
		ID:     hostID,
		Name:   hostName,
		IsHost: true,
		JoinedAt: time.Now(),
	}

	return &Session{
		ID:           generateID(),
		Code:         code,
		Phase:        PhaseJoining,
		Participants: map[string]*Participant{hostID: host},
		Notes:        []*Note{},
		CreatedAt:    time.Now(),
		HostID:       hostID,
		CurrentTurn:  0,
	}
}

// AddParticipant adds a new participant to the session
func (s *Session) AddParticipant(name string) (*Participant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Phase != PhaseJoining {
		return nil, errors.New("cannot join: session has already started")
	}

	participant := &Participant{
		ID:     generateID(),
		Name:   name,
		IsHost: false,
		JoinedAt: time.Now(),
	}

	s.Participants[participant.ID] = participant
	return participant, nil
}

// AddNote adds a gratitude note to the session
func (s *Session) AddNote(authorID, recipientID, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Phase != PhaseWriting {
		return errors.New("cannot add note: not in writing phase")
	}

	// Validate author exists
	if _, exists := s.Participants[authorID]; !exists {
		return errors.New("author not found in session")
	}

	// Validate recipient exists
	if _, exists := s.Participants[recipientID]; !exists {
		return errors.New("recipient not found in session")
	}

	// Cannot write to self
	if authorID == recipientID {
		return errors.New("cannot write note to yourself")
	}

	// Check if note already exists from this author to this recipient
	for _, note := range s.Notes {
		if note.AuthorID == authorID && note.RecipientID == recipientID {
			return errors.New("note already written to this person")
		}
	}

	note := &Note{
		ID:          generateID(),
		Content:     content,
		AuthorID:    authorID,
		RecipientID: recipientID,
		Read:        false,
	}

	s.Notes = append(s.Notes, note)
	return nil
}

// TransitionToWriting moves the session to writing phase
func (s *Session) TransitionToWriting() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Phase != PhaseJoining {
		return errors.New("can only transition to writing from joining phase")
	}

	if len(s.Participants) < 2 {
		return errors.New("need at least 2 participants to start")
	}

	s.Phase = PhaseWriting
	return nil
}

// TransitionToReading moves the session to reading phase
func (s *Session) TransitionToReading() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Phase != PhaseWriting {
		return errors.New("can only transition to reading from writing phase")
	}

	// Verify all notes have been written
	expectedNotes := len(s.Participants) * (len(s.Participants) - 1)
	if len(s.Notes) != expectedNotes {
		return errors.New("not all notes have been written")
	}

	s.Phase = PhaseReading
	return nil
}

// GetUnreadNotes returns notes that haven't been read yet
func (s *Session) GetUnreadNotes() []*Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	unread := []*Note{}
	for _, note := range s.Notes {
		if !note.Read {
			unread = append(unread, note)
		}
	}
	return unread
}

// GetAvailableNotesForReader returns notes that the reader can read
// (not authored by them, and in 3+ person sessions, not addressed to them)
// Note: In 2-person sessions, readers CAN read notes written to them
// since there's no one else to do it
func (s *Session) GetAvailableNotesForReader(readerID string) []*Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getAvailableNotesForReaderUnlocked(readerID)
}

// getAvailableNotesForReaderUnlocked returns notes that the reader can read
// Internal helper that assumes caller already holds a lock
func (s *Session) getAvailableNotesForReaderUnlocked(readerID string) []*Note {
	available := []*Note{}
	participantCount := len(s.Participants)

	for _, note := range s.Notes {
		// Skip notes already read
		if note.Read {
			continue
		}

		// Never read notes you authored
		if note.AuthorID == readerID {
			continue
		}

		// In 3+ person sessions, don't read notes addressed to you
		// (preserves surprise - someone else should read them to you)
		if participantCount > 2 && note.RecipientID == readerID {
			continue
		}

		available = append(available, note)
	}
	return available
}

// MarkNoteAsRead marks a note as read
func (s *Session) MarkNoteAsRead(noteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, note := range s.Notes {
		if note.ID == noteID {
			note.Read = true
			return nil
		}
	}

	return errors.New("note not found")
}

// GetCurrentReader returns the participant whose turn it is to read
func (s *Session) GetCurrentReader() *Participant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Phase != PhaseReading {
		return nil
	}

	// Get participants in stable sorted order by ID
	participants := s.getParticipantsSorted()

	if len(participants) == 0 {
		return nil
	}

	return participants[s.CurrentTurn%len(participants)]
}

// AdvanceTurn moves to the next reader
// Intelligently skips readers who have no available notes to draw
func (s *Session) AdvanceTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()

	participants := s.getParticipantsSorted()
	if len(participants) == 0 {
		return
	}

	// Try to find the next reader with available notes
	// Limit iterations to prevent infinite loops
	maxAttempts := len(participants)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		s.CurrentTurn++

		// Get the current reader based on new turn value
		currentReaderIndex := s.CurrentTurn % len(participants)
		currentReader := participants[currentReaderIndex]

		// Check if this reader has any available notes
		availableNotes := s.getAvailableNotesForReaderUnlocked(currentReader.ID)
		if len(availableNotes) > 0 {
			// Found a reader with available notes
			return
		}

		// No available notes for this reader, continue to next
	}

	// If we've cycled through all participants and nobody has available notes,
	// check if all notes are actually read
	allRead := true
	for _, note := range s.Notes {
		if !note.Read {
			allRead = false
			break
		}
	}

	if allRead {
		s.Phase = PhaseComplete
	} else {
		// Deadlock scenario: unread notes exist but nobody can read them
		// This shouldn't happen with proper note filtering, but handle gracefully
		s.Phase = PhaseComplete
	}
}

// RemoveParticipant removes a participant from the session
func (s *Session) RemoveParticipant(participantID string) (*Participant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant, exists := s.Participants[participantID]
	if !exists {
		return nil, errors.New("participant not found")
	}

	delete(s.Participants, participantID)
	return participant, nil
}

// HasParticipant checks if a participant is in the session
func (s *Session) HasParticipant(participantID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.Participants[participantID]
	return exists
}

// GetParticipantList returns a slice of all participants
func (s *Session) GetParticipantList() []*Participant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	participants := make([]*Participant, 0, len(s.Participants))
	for _, p := range s.Participants {
		participants = append(participants, p)
	}
	return participants
}

// getParticipantsSorted returns participants in stable sorted order by ID
// This ensures consistent turn order across all function calls
// Note: This is an internal helper and assumes caller already holds a lock
func (s *Session) getParticipantsSorted() []*Participant {
	participants := make([]*Participant, 0, len(s.Participants))
	for _, p := range s.Participants {
		participants = append(participants, p)
	}

	// Sort by ID to ensure stable ordering
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].ID < participants[j].ID
	})

	return participants
}

// generateSessionCode generates a short, memorable session code
func generateSessionCode() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	code := base32.StdEncoding.EncodeToString(b)
	// Remove padding and limit to 6 characters
	code = strings.TrimRight(code, "=")
	if len(code) > 6 {
		code = code[:6]
	}
	return code
}

// generateID generates a unique identifier
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base32.StdEncoding.EncodeToString(b)
}
