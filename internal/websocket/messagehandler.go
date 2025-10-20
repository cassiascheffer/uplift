// ABOUTME: Message handler for processing WebSocket messages and coordinating session operations
// ABOUTME: Bridges WebSocket communication with session management logic
package websocket

import (
	"log"
	"math/rand"

	"github.com/cassiascheffer/uplift/internal/session"
)

// MessageHandler handles incoming WebSocket messages
type MessageHandler struct {
	hub            *Hub
	sessionManager *session.Manager
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(hub *Hub, sessionManager *session.Manager) *MessageHandler {
	return &MessageHandler{
		hub:            hub,
		sessionManager: sessionManager,
	}
}

// HandleMessage processes an incoming message from a client
func (mh *MessageHandler) HandleMessage(client *Client, msg *Message) {
	log.Printf("HandleMessage: type=%s sessionID=%s userID=%s", msg.Type, client.sessionID, client.userID)
	switch msg.Type {
	case "validate_session":
		mh.handleValidateSession(client, msg)
	case "create_session":
		mh.handleCreateSession(client, msg)
	case "join_session":
		mh.handleJoinSession(client, msg)
	case "start_writing":
		mh.handleStartWriting(client, msg)
	case "submit_notes":
		mh.handleSubmitNotes(client, msg)
	case "draw_note":
		mh.handleDrawNote(client, msg)
	case "note_read":
		mh.handleNoteRead(client, msg)
	case "remove_participant":
		mh.handleRemoveParticipant(client, msg)
	default:
		log.Printf("unknown message type: %s", msg.Type)
	}
}

// HandleClientDisconnect processes a client disconnection
func (mh *MessageHandler) HandleClientDisconnect(client *Client) {
	if client.sessionID == "" || client.userID == "" {
		return // Client never joined a session
	}

	log.Printf("HandleClientDisconnect: sessionID=%s userID=%s", client.sessionID, client.userID)

	// Get session
	sess, err := mh.sessionManager.GetSessionByID(client.sessionID)
	if err != nil {
		log.Printf("Session not found for disconnecting client: %v", err)
		return
	}

	// Check if this was the host
	wasHost := client.userID == sess.HostID

	// Remove participant from session
	participant, err := sess.RemoveParticipant(client.userID)
	if err != nil {
		log.Printf("Error removing participant: %v", err)
		return
	}

	// If host left and there are participants remaining, assign new host
	if wasHost && len(sess.Participants) > 0 {
		// Get first remaining participant as new host
		for _, p := range sess.Participants {
			p.IsHost = true
			sess.HostID = p.ID
			log.Printf("New host assigned: session=%s userId=%s", sess.Code, p.ID)
			break
		}
	}

	// Check if session is now empty
	if len(sess.Participants) == 0 {
		// Remove session from manager
		if err := mh.sessionManager.RemoveSession(sess.ID); err != nil {
			log.Printf("Error removing empty session: %v", err)
		} else {
			log.Printf("Empty session cleaned up: session=%s", sess.Code)
		}
		return
	}

	// Broadcast participant left to remaining clients
	broadcast := &Message{
		Type: "participant_left",
		Data: map[string]interface{}{
			"participant":  participant,
			"participants": sess.GetParticipantList(),
			"wasHost":      wasHost,
		},
	}
	mh.hub.BroadcastToSession(sess.ID, broadcast)

	log.Printf("Participant removed from session: session=%s userId=%s wasHost=%v", sess.Code, participant.ID, wasHost)
}

// handleValidateSession validates if a session code exists without joining
func (mh *MessageHandler) handleValidateSession(client *Client, msg *Message) {
	sessionCode, ok := msg.Data["sessionCode"].(string)
	if !ok || sessionCode == "" {
		response := &Message{
			Type: "session_validation",
			Data: map[string]interface{}{
				"valid": false,
				"error": "session code required",
			},
		}
		client.SendMessage(response)
		return
	}

	// Check if session exists
	_, err := mh.sessionManager.GetSessionByCode(sessionCode)
	if err != nil {
		response := &Message{
			Type: "session_validation",
			Data: map[string]interface{}{
				"valid": false,
				"error": "session not found",
			},
		}
		client.SendMessage(response)
		log.Printf("Session validation failed: code=%s", sessionCode)
		return
	}

	// Session exists
	response := &Message{
		Type: "session_validation",
		Data: map[string]interface{}{
			"valid": true,
		},
	}
	client.SendMessage(response)
	log.Printf("Session validated: code=%s", sessionCode)
}

// handleCreateSession creates a new session
func (mh *MessageHandler) handleCreateSession(client *Client, msg *Message) {
	userName, ok := msg.Data["userName"].(string)
	if !ok || userName == "" {
		userName = "Host"
	}

	// Create session
	sess := mh.sessionManager.CreateSession(userName)

	// Get the host participant (first and only participant)
	participants := sess.GetParticipantList()
	if len(participants) == 0 {
		mh.sendError(client, "failed to create session")
		return
	}
	host := participants[0]

	// Associate client with session
	client.sessionID = sess.ID
	client.userID = host.ID
	client.userName = host.Name

	// Register client with hub now that we have sessionID
	// Use goroutine to avoid blocking the hub's Run loop
	go func() {
		mh.hub.register <- client
	}()

	// Send confirmation to client
	response := &Message{
		Type: "session_created",
		Data: map[string]interface{}{
			"sessionCode":  sess.Code,
			"sessionId":    sess.ID,
			"userId":       host.ID,
			"userName":     host.Name,
			"participants": participants,
			"phase":        sess.Phase,
		},
	}
	client.SendMessage(response)

	log.Printf("Session created: code=%s id=%s", sess.Code, sess.ID)
}

// handleJoinSession joins an existing session
func (mh *MessageHandler) handleJoinSession(client *Client, msg *Message) {
	sessionCode, ok := msg.Data["sessionCode"].(string)
	if !ok || sessionCode == "" {
		mh.sendError(client, "session code required")
		return
	}

	userName, ok := msg.Data["userName"].(string)
	if !ok || userName == "" {
		mh.sendError(client, "user name required")
		return
	}

	// Get session by code
	sess, err := mh.sessionManager.GetSessionByCode(sessionCode)
	if err != nil {
		mh.sendError(client, "session not found")
		return
	}

	// Add participant to session
	participant, err := sess.AddParticipant(userName)
	if err != nil {
		mh.sendError(client, err.Error())
		return
	}

	// Associate client with session
	client.sessionID = sess.ID
	client.userID = participant.ID
	client.userName = participant.Name

	// Register client with hub now that we have sessionID
	// Use goroutine to avoid blocking the hub's Run loop
	go func() {
		mh.hub.register <- client
	}()

	// Send confirmation to joining client
	response := &Message{
		Type: "session_joined",
		Data: map[string]interface{}{
			"sessionCode":  sess.Code,
			"sessionId":    sess.ID,
			"userId":       participant.ID,
			"userName":     participant.Name,
			"participants": sess.GetParticipantList(),
			"phase":        sess.Phase,
		},
	}
	client.SendMessage(response)

	// Broadcast participant joined to all other clients
	broadcast := &Message{
		Type: "participant_joined",
		Data: map[string]interface{}{
			"participant":  participant,
			"participants": sess.GetParticipantList(),
		},
	}
	mh.hub.BroadcastToSessionExcept(sess.ID, participant.ID, broadcast)

	log.Printf("Participant joined: session=%s userId=%s", sess.Code, participant.ID)
}

// handleStartWriting transitions session to writing phase
func (mh *MessageHandler) handleStartWriting(client *Client, msg *Message) {
	log.Printf("handleStartWriting: sessionID=%s userID=%s", client.sessionID, client.userID)

	sess, err := mh.sessionManager.GetSessionByID(client.sessionID)
	if err != nil {
		log.Printf("Error getting session: %v", err)
		mh.sendError(client, "session not found")
		return
	}

	log.Printf("Session found: %s, HostID=%s, ClientUserID=%s", sess.Code, sess.HostID, client.userID)

	// Verify client is host
	if client.userID != sess.HostID {
		log.Printf("User is not host: userID=%s hostID=%s", client.userID, sess.HostID)
		mh.sendError(client, "only host can start writing phase")
		return
	}

	// Transition to writing phase
	if err := sess.TransitionToWriting(); err != nil {
		mh.sendError(client, err.Error())
		return
	}

	// Broadcast phase change to all clients
	broadcast := &Message{
		Type: "phase_changed",
		Data: map[string]interface{}{
			"phase":             sess.Phase,
			"participants":      sess.GetParticipantList(),
			"totalNotesNeeded": len(sess.Participants) - 1,
		},
	}
	mh.hub.BroadcastToSession(sess.ID, broadcast)

	log.Printf("Writing phase started: session=%s", sess.Code)
}

// handleSubmitNotes processes submitted gratitude notes
func (mh *MessageHandler) handleSubmitNotes(client *Client, msg *Message) {
	sess, err := mh.sessionManager.GetSessionByID(client.sessionID)
	if err != nil {
		mh.sendError(client, "session not found")
		return
	}

	notes, ok := msg.Data["notes"].([]interface{})
	if !ok {
		mh.sendError(client, "invalid notes format")
		return
	}

	// Add each note to the session
	for _, noteData := range notes {
		noteMap, ok := noteData.(map[string]interface{})
		if !ok {
			continue
		}

		recipientID, ok := noteMap["recipientId"].(string)
		if !ok {
			continue
		}

		content, ok := noteMap["content"].(string)
		if !ok || content == "" {
			continue
		}

		if err := sess.AddNote(client.userID, recipientID, content); err != nil {
			log.Printf("error adding note: %v", err)
			mh.sendError(client, err.Error())
			return
		}
	}

	// Send confirmation
	response := &Message{
		Type: "notes_submitted",
		Data: map[string]interface{}{
			"success": true,
		},
	}
	client.SendMessage(response)

	// Check if all notes have been submitted
	expectedNotes := len(sess.Participants) * (len(sess.Participants) - 1)
	if len(sess.Notes) == expectedNotes {
		// Automatically transition to reading phase
		if err := sess.TransitionToReading(); err != nil {
			log.Printf("error transitioning to reading: %v", err)
			return
		}

		// Broadcast phase change
		currentReader := sess.GetCurrentReader()
		broadcast := &Message{
			Type: "phase_changed",
			Data: map[string]interface{}{
				"phase":         sess.Phase,
				"currentReader": currentReader,
			},
		}
		mh.hub.BroadcastToSession(sess.ID, broadcast)

		log.Printf("Reading phase started: session=%s", sess.Code)
	}
}

// handleDrawNote draws a random note for the current reader
func (mh *MessageHandler) handleDrawNote(client *Client, msg *Message) {
	sess, err := mh.sessionManager.GetSessionByID(client.sessionID)
	if err != nil {
		mh.sendError(client, "session not found")
		return
	}

	// Verify it's the client's turn
	currentReader := sess.GetCurrentReader()
	if currentReader == nil || currentReader.ID != client.userID {
		mh.sendError(client, "not your turn")
		return
	}

	// Get available notes (not authored by or for the reader)
	availableNotes := sess.GetAvailableNotesForReader(client.userID)
	if len(availableNotes) == 0 {
		// Current reader has no available notes - auto-advance turn
		log.Printf("No available notes for reader: session=%s readerId=%s, auto-advancing turn", sess.Code, client.userID)
		sess.AdvanceTurn()

		// Check if session is complete
		if sess.Phase == session.PhaseComplete {
			// Prepare notes (anonymous - no author names)
			anonymousNotes := []map[string]interface{}{}
			for _, note := range sess.Notes {
				anonymousNotes = append(anonymousNotes, map[string]interface{}{
					"id":          note.ID,
					"content":     note.Content,
					"recipientId": note.RecipientID,
				})
			}

			broadcast := &Message{
				Type: "session_complete",
				Data: map[string]interface{}{
					"message": "All notes have been read. Thank you for participating!",
					"notes":   anonymousNotes,
				},
			}
			mh.hub.BroadcastToSession(sess.ID, broadcast)
			log.Printf("Session complete: session=%s", sess.Code)
			return
		}

		// Broadcast turn change to all clients
		newReader := sess.GetCurrentReader()
		unreadNotes := sess.GetUnreadNotes()
		totalNotes := len(sess.Notes)
		broadcast := &Message{
			Type: "turn_changed",
			Data: map[string]interface{}{
				"reader":    newReader,
				"remaining": len(unreadNotes),
				"total":     totalNotes,
			},
		}
		mh.hub.BroadcastToSession(sess.ID, broadcast)
		log.Printf("Turn auto-advanced: session=%s newReaderId=%s", sess.Code, newReader.ID)
		return
	}

	// Pick a random note
	randomNote := availableNotes[rand.Intn(len(availableNotes))]

	// Get recipient name
	var recipientName string
	if recipient, exists := sess.Participants[randomNote.RecipientID]; exists {
		recipientName = recipient.Name
	}

	// Send note to all clients
	unreadNotes := sess.GetUnreadNotes()
	totalNotes := len(sess.Notes)
	broadcast := &Message{
		Type: "note_drawn",
		Data: map[string]interface{}{
			"note": map[string]interface{}{
				"id":        randomNote.ID,
				"content":   randomNote.Content,
				"recipient": recipientName,
			},
			"remaining": len(unreadNotes) - 1,
			"total":     totalNotes,
		},
	}
	mh.hub.BroadcastToSession(sess.ID, broadcast)

	log.Printf("Note drawn: session=%s readerId=%s", sess.Code, client.userID)
}

// handleNoteRead marks the current note as read and advances turn
func (mh *MessageHandler) handleNoteRead(client *Client, msg *Message) {
	sess, err := mh.sessionManager.GetSessionByID(client.sessionID)
	if err != nil {
		mh.sendError(client, "session not found")
		return
	}

	// Verify it's the client's turn
	currentReader := sess.GetCurrentReader()
	if currentReader == nil || currentReader.ID != client.userID {
		mh.sendError(client, "not your turn")
		return
	}

	// Get the note ID from the message
	noteID, ok := msg.Data["noteId"].(string)
	if !ok {
		// If no noteID provided, we can't mark it as read
		// This shouldn't happen but we'll handle it gracefully
		log.Printf("no noteId provided in note_read message")
	} else {
		// Mark note as read
		if err := sess.MarkNoteAsRead(noteID); err != nil {
			log.Printf("error marking note as read: %v", err)
		}
	}

	// Advance turn
	sess.AdvanceTurn()

	// Check if session is complete
	if sess.Phase == session.PhaseComplete {
		// Prepare notes (anonymous - no author names)
		anonymousNotes := []map[string]interface{}{}
		for _, note := range sess.Notes {
			anonymousNotes = append(anonymousNotes, map[string]interface{}{
				"id":          note.ID,
				"content":     note.Content,
				"recipientId": note.RecipientID,
			})
		}

		broadcast := &Message{
			Type: "session_complete",
			Data: map[string]interface{}{
				"message": "All notes have been read. Thank you for participating!",
				"notes":   anonymousNotes,
			},
		}
		mh.hub.BroadcastToSession(sess.ID, broadcast)
		log.Printf("Session complete: session=%s", sess.Code)
		return
	}

	// Send turn change to all clients
	newReader := sess.GetCurrentReader()
	unreadNotes := sess.GetUnreadNotes()
	totalNotes := len(sess.Notes)
	broadcast := &Message{
		Type: "turn_changed",
		Data: map[string]interface{}{
			"reader":    newReader,
			"remaining": len(unreadNotes),
			"total":     totalNotes,
		},
	}
	mh.hub.BroadcastToSession(sess.ID, broadcast)

	log.Printf("Turn advanced: session=%s newReaderId=%s", sess.Code, newReader.ID)
}

// handleRemoveParticipant removes a participant from the session (host only)
func (mh *MessageHandler) handleRemoveParticipant(client *Client, msg *Message) {
	sess, err := mh.sessionManager.GetSessionByID(client.sessionID)
	if err != nil {
		mh.sendError(client, "session not found")
		return
	}

	// Verify client is host
	if client.userID != sess.HostID {
		log.Printf("Non-host tried to remove participant: userID=%s hostID=%s", client.userID, sess.HostID)
		mh.sendError(client, "only host can remove participants")
		return
	}

	// Get participant ID to remove
	participantID, ok := msg.Data["participantId"].(string)
	if !ok || participantID == "" {
		mh.sendError(client, "participant ID required")
		return
	}

	// Cannot remove yourself
	if participantID == client.userID {
		mh.sendError(client, "cannot remove yourself")
		return
	}

	// Remove participant from session
	participant, err := sess.RemoveParticipant(participantID)
	if err != nil {
		mh.sendError(client, err.Error())
		return
	}

	// Send kicked message to the removed user
	kickedMsg := &Message{
		Type: "kicked",
		Data: map[string]interface{}{
			"message": "You have been removed from the session by the host",
		},
	}
	mh.hub.SendToUser(sess.ID, participantID, kickedMsg)

	// Broadcast participant left to remaining clients
	broadcast := &Message{
		Type: "participant_left",
		Data: map[string]interface{}{
			"participant":  participant,
			"participants": sess.GetParticipantList(),
			"wasHost":      false,
			"wasRemoved":   true,
		},
	}
	mh.hub.BroadcastToSession(sess.ID, broadcast)

	log.Printf("Participant removed by host: session=%s userId=%s", sess.Code, participant.ID)
}

// sendError sends an error message to a client
func (mh *MessageHandler) sendError(client *Client, message string) {
	response := &Message{
		Type: "error",
		Data: map[string]interface{}{
			"message": message,
		},
	}
	client.SendMessage(response)
	log.Printf("Error sent to client: %s", message)
}
