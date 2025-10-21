// ABOUTME: Input validation and sanitisation for WebSocket messages
// ABOUTME: Prevents memory exhaustion and UI breaking from excessive input
package websocket

import (
	"errors"
	"strings"
)

const (
	maxUserNameLength = 100
	maxNoteLength     = 2000
	maxParticipants   = 50
)

var (
	ErrUserNameEmpty    = errors.New("user name cannot be empty")
	ErrUserNameTooLong  = errors.New("user name too long (max 100 characters)")
	ErrNoteEmpty        = errors.New("note content cannot be empty")
	ErrNoteTooLong      = errors.New("note content too long (max 2000 characters)")
	ErrTooManyParticipants = errors.New("session is full (max 50 participants)")
)

// validateUserName validates and sanitises a user name
func validateUserName(name string) (string, error) {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Check if empty
	if name == "" {
		return "", ErrUserNameEmpty
	}

	// Check length
	if len(name) > maxUserNameLength {
		return "", ErrUserNameTooLong
	}

	return name, nil
}

// validateNoteContent validates and sanitises note content
func validateNoteContent(content string) (string, error) {
	// Trim whitespace
	content = strings.TrimSpace(content)

	// Check if empty
	if content == "" {
		return "", ErrNoteEmpty
	}

	// Check length
	if len(content) > maxNoteLength {
		return "", ErrNoteTooLong
	}

	return content, nil
}

// checkParticipantLimit checks if session has reached max participants
func checkParticipantLimit(currentCount int) error {
	if currentCount >= maxParticipants {
		return ErrTooManyParticipants
	}
	return nil
}
