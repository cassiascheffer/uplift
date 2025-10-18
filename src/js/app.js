// ABOUTME: Alpine.js application logic for uplift frontend
// ABOUTME: Manages client state, WebSocket communication, and UI interactions

function uplift() {
    return {
        // Connection state
        ws: null,
        connected: false,
        isConnecting: false,
        reconnectAttempts: 0,
        maxReconnectDelay: 30000, // 30 seconds max

        // View state
        currentView: 'home', // home, create, join, lobby, writing, reading
        onboardingStep: 'choice', // choice, name_entry
        fromDirectLink: false,

        // Session state
        sessionCode: '',
        isHost: false,
        myId: null,
        userName: '',
        joinCode: '',
        selectedAction: null, // 'create' or 'join'

        // Participants
        participants: [],

        // Writing phase
        notes: {},
        notesWritten: 0,
        totalNotesNeeded: 0,
        allNotesWritten: false,
        currentNoteIndex: 0, // Track which participant we're writing for
        hasSubmittedNotes: false, // Track if user has submitted

        // Reading phase
        currentReader: null,
        currentNote: null,
        notesRemaining: 0,
        totalNotes: 0,
        isMyTurn: false,
        sessionComplete: false,
        animateNote: false,

        // Completion
        receivedNotes: [],


        // Notifications
        notifications: [],
        recentlyJoinedIds: new Set(),

        // Screen reader announcements
        srAnnouncement: '',

        // Host controls
        participantToRemove: null,

        init() {
            console.log('Uplift initialized');
            this.loadTheme();
            this.setupBeforeUnload();
            this.checkForSessionCodeInURL();
        },

        checkForSessionCodeInURL() {
            const urlParams = new URLSearchParams(window.location.search);
            const codeFromURL = urlParams.get('code');
            if (codeFromURL) {
                this.joinCode = codeFromURL.toUpperCase();
                this.selectedAction = 'join';
                this.onboardingStep = 'name_entry';
                this.fromDirectLink = true;
                console.log('Pre-filled join code from URL:', this.joinCode);
            }
        },

        setupBeforeUnload() {
            window.addEventListener('beforeunload', (e) => {
                // Only warn if user is in an active session (not home view)
                if (this.sessionCode && this.currentView !== 'home') {
                    e.preventDefault();
                    e.returnValue = '';
                }
            });
        },

        loadTheme() {
            // Detect system preference for dark mode
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const theme = prefersDark ? 'halloween' : 'bumblebee';
            document.documentElement.setAttribute('data-theme', theme);

            // Listen for system theme changes
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
                const newTheme = e.matches ? 'halloween' : 'bumblebee';
                document.documentElement.setAttribute('data-theme', newTheme);
            });
        },

        connectWebSocket(onConnected) {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws`;

            console.log('Attempting WebSocket connection to:', wsUrl);
            this.isConnecting = true;
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                console.log('WebSocket connected successfully');
                this.connected = true;
                this.isConnecting = false;

                // Show reconnected message if this was a reconnection
                if (this.reconnectAttempts > 0) {
                    this.showNotification('Reconnected successfully!');
                    this.reconnectAttempts = 0;
                }

                if (onConnected) {
                    console.log('Calling onConnected callback');
                    onConnected();
                }
            };

            this.ws.onmessage = (event) => {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error occurred:', error);
                console.error('WebSocket readyState:', this.ws?.readyState);
                this.isConnecting = false;
                if (this.sessionCode) {
                    this.showNotification('Connection error. Attempting to reconnect...', 'error');
                }
            };

            this.ws.onclose = (event) => {
                console.log('WebSocket closed. Code:', event.code, 'Reason:', event.reason, 'Clean:', event.wasClean);
                this.connected = false;
                this.isConnecting = false;

                // Only attempt reconnection if in an active session and not a clean close
                if (this.sessionCode && !event.wasClean) {
                    this.reconnectAttempts++;
                    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
                    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts - 1), this.maxReconnectDelay);
                    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
                    this.showNotification(`Disconnected. Reconnecting in ${Math.ceil(delay/1000)}s...`, 'error');
                    // Attempt to reconnect with exponential backoff
                    setTimeout(() => this.connectWebSocket(), delay);
                }
            };
        },

        handleMessage(message) {
            console.log('Received message:', message);

            switch (message.type) {
                case 'session_created':
                    this.sessionCode = message.data.sessionCode;
                    this.myId = message.data.userId;
                    this.isHost = true;
                    this.participants = message.data.participants;
                    this.currentView = 'lobby';
                    break;

                case 'session_joined':
                    this.sessionCode = message.data.sessionCode;
                    this.myId = message.data.userId;
                    this.participants = message.data.participants;
                    this.currentView = 'lobby';
                    break;

                case 'participant_joined':
                    const newParticipants = message.data.participants;
                    // Find who just joined by comparing participant lists
                    if (this.participants.length > 0) {
                        const existingIds = new Set(this.participants.map(p => p.id));
                        const newParticipant = newParticipants.find(p => !existingIds.has(p.id));
                        if (newParticipant && newParticipant.id !== this.myId) {
                            this.showNotification(`${newParticipant.name} arrived!`);
                            this.recentlyJoinedIds.add(newParticipant.id);
                            // Remove 'new' indicator after 5 seconds
                            setTimeout(() => {
                                this.recentlyJoinedIds.delete(newParticipant.id);
                            }, 5000);
                        }
                    }
                    this.participants = newParticipants;
                    break;

                case 'participant_left':
                    const leftParticipant = message.data.participant;
                    const wasHostLeaving = message.data.wasHost;
                    const wasRemoved = message.data.wasRemoved;
                    if (leftParticipant && leftParticipant.id !== this.myId) {
                        if (wasHostLeaving) {
                            this.showNotification(`${leftParticipant.name} (host) left. New host assigned.`);
                        } else if (wasRemoved) {
                            this.showNotification(`${leftParticipant.name} was removed from the session`);
                        } else {
                            this.showNotification(`${leftParticipant.name} left the session`);
                        }
                    }
                    this.participants = message.data.participants;
                    // Update isHost status if you became the new host
                    const myParticipant = this.participants.find(p => p.id === this.myId);
                    if (myParticipant) {
                        this.isHost = myParticipant.isHost;
                    }
                    break;

                case 'kicked':
                    this.showNotification(message.data.message, 'error');
                    // Close WebSocket and return to home
                    if (this.ws) {
                        this.ws.close();
                        this.ws = null;
                    }
                    this.sessionCode = '';
                    this.currentView = 'home';
                    break;

                case 'timeout':
                    this.showNotification(message.data.message, 'error');
                    // Close WebSocket and return to home
                    if (this.ws) {
                        this.ws.close();
                        this.ws = null;
                    }
                    this.sessionCode = '';
                    this.currentView = 'home';
                    break;

                case 'phase_changed':
                    console.log('Phase changed received:', message.data);
                    this.handlePhaseChange(message.data);
                    break;

                case 'notes_submitted':
                    // Handle notes submission confirmation
                    break;

                case 'turn_changed':
                    this.currentReader = message.data.reader;
                    this.isMyTurn = message.data.reader.id === this.myId;
                    this.currentNote = null; // Clear note from all screens when turn changes
                    if (this.isMyTurn) {
                        this.announceToScreenReader("It's your turn to pick a note");
                    } else {
                        this.announceToScreenReader(`${this.currentReader.name} is now reading`);
                    }
                    break;

                case 'note_drawn':
                    this.currentNote = message.data.note;
                    this.notesRemaining = message.data.remaining;
                    // Set totalNotes if not already set
                    if (this.totalNotes === 0 && message.data.total) {
                        this.totalNotes = message.data.total;
                    }
                    // Trigger animation
                    this.animateNote = true;
                    setTimeout(() => { this.animateNote = false; }, 600);
                    this.announceToScreenReader(`Note picked for ${this.currentNote.recipient}`);
                    break;

                case 'session_complete':
                    this.sessionComplete = true;
                    // Filter notes to show only those received by this user
                    if (message.data.notes) {
                        this.receivedNotes = message.data.notes.filter(note => note.recipientId === this.myId);
                    }
                    this.announceToScreenReader('Session complete! All notes have been read.');
                    break;

                case 'error':
                    this.showNotification(message.data.message, 'error');
                    break;
            }
        },

        handlePhaseChange(data) {
            const phase = data.phase;

            switch (phase) {
                case 'WRITING':
                    this.currentView = 'writing';
                    this.totalNotesNeeded = this.participants.length - 1;
                    this.currentNoteIndex = 0;
                    this.hasSubmittedNotes = false;
                    this.updateNotesProgress();
                    this.announceToScreenReader('Writing phase started. Write a note of appreciation for each person.');
                    break;

                case 'READING':
                    this.currentView = 'reading';
                    if (data.currentReader) {
                        this.currentReader = data.currentReader;
                        this.isMyTurn = data.currentReader.id === this.myId;
                        console.log('Reading phase: currentReader=', this.currentReader, 'isMyTurn=', this.isMyTurn);
                    }
                    // Calculate total notes: each participant writes to everyone else
                    if (this.totalNotes === 0) {
                        const participantCount = this.participants.length;
                        this.totalNotes = participantCount * (participantCount - 1);
                    }
                    // Initialize notesRemaining to totalNotes when starting reading phase
                    if (this.notesRemaining === 0) {
                        this.notesRemaining = this.totalNotes;
                    }
                    this.announceToScreenReader('Reading phase started. Take turns picking and reading notes aloud.');
                    break;

                case 'COMPLETE':
                    this.sessionComplete = true;
                    this.announceToScreenReader('Session complete!');
                    break;
            }
        },

        updateNotesProgress() {
            let count = 0;
            for (const [participantId, content] of Object.entries(this.notes)) {
                if (content && content.trim()) {
                    count++;
                }
            }
            this.notesWritten = count;
        },

        send(message) {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                console.log('Sending message:', message);
                this.ws.send(JSON.stringify(message));
            } else {
                console.error('WebSocket not connected, state:', this.ws?.readyState);
            }
        },

        createSession() {
            if (!this.userName || !this.userName.trim()) {
                this.showNotification('Please enter your name', 'error');
                return;
            }

            console.log('createSession: WebSocket state:', this.ws?.readyState);

            // If already connected, just send the message
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                console.log('Already connected, sending message directly');
                this.send({
                    type: 'create_session',
                    data: {
                        userName: this.userName.trim()
                    }
                });
                return;
            }

            console.log('Initiating new WebSocket connection...');
            this.connectWebSocket(() => {
                this.send({
                    type: 'create_session',
                    data: {
                        userName: this.userName.trim()
                    }
                });
            });
        },

        joinSession() {
            if (!this.joinCode || !this.userName) {
                this.showNotification('Please enter both session code and your name', 'error');
                return;
            }

            console.log('joinSession: WebSocket state:', this.ws?.readyState);

            // If already connected, just send the message
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                console.log('Already connected, sending message directly');
                this.send({
                    type: 'join_session',
                    data: {
                        sessionCode: this.joinCode.toUpperCase(),
                        userName: this.userName
                    }
                });
                return;
            }

            console.log('Initiating new WebSocket connection...');
            this.connectWebSocket(() => {
                this.send({
                    type: 'join_session',
                    data: {
                        sessionCode: this.joinCode.toUpperCase(),
                        userName: this.userName
                    }
                });
            });
        },

        startWriting() {
            console.log('startWriting called, isHost:', this.isHost);
            if (!this.isHost) {
                console.log('Not host, cannot start writing phase');
                return;
            }

            console.log('Starting writing phase...');
            this.send({
                type: 'start_writing',
                data: {}
            });
        },

        // Get the list of participants to write notes for (excluding self)
        getRecipients() {
            return this.participants.filter(p => p.id !== this.myId);
        },

        // Get the current recipient being written for
        getCurrentRecipient() {
            const recipients = this.getRecipients();
            return recipients[this.currentNoteIndex] || null;
        },

        // Check if this is the last note
        isLastNote() {
            return this.currentNoteIndex >= this.getRecipients().length - 1;
        },

        // Move to next note
        nextNote() {
            const currentRecipient = this.getCurrentRecipient();
            if (!currentRecipient) return;

            // Save current note (even if empty)
            const currentContent = this.notes[currentRecipient.id] || '';

            if (this.isLastNote()) {
                // This is the last note - submit all notes
                this.submitNotes();
            } else {
                // Move to next recipient
                this.currentNoteIndex++;
                this.announceToScreenReader(`Writing note for ${this.getCurrentRecipient()?.name}`);
            }
        },

        // Go back to previous note
        previousNote() {
            if (this.currentNoteIndex > 0) {
                this.currentNoteIndex--;
                this.announceToScreenReader(`Writing note for ${this.getCurrentRecipient()?.name}`);
            }
        },

        submitNotes() {
            const notesList = [];
            for (const [recipientId, content] of Object.entries(this.notes)) {
                if (content && content.trim()) {
                    notesList.push({
                        recipientId,
                        content: content.trim()
                    });
                }
            }

            this.send({
                type: 'submit_notes',
                data: {
                    notes: notesList
                }
            });

            // Mark as submitted and show waiting state
            this.hasSubmittedNotes = true;
            this.announceToScreenReader('Notes submitted. Waiting for others to finish writing.');
        },

        drawNote() {
            if (!this.isMyTurn) return;

            this.send({
                type: 'draw_note'
            });
        },

        markNoteRead() {
            if (!this.isMyTurn) return;

            this.send({
                type: 'note_read',
                data: {
                    noteId: this.currentNote?.id
                }
            });

            // Clear current note
            this.currentNote = null;
        },

        selectAction(action) {
            this.selectedAction = action;
            this.onboardingStep = 'name_entry';
        },

        goBackToChoice() {
            this.onboardingStep = 'choice';
            this.selectedAction = null;
            this.userName = '';
            this.joinCode = '';
            this.fromDirectLink = false;
        },

        leaveSession() {
            // Close WebSocket connection
            if (this.ws) {
                this.ws.close();
                this.ws = null;
            }

            // Reset all state
            this.sessionCode = '';
            this.isHost = false;
            this.myId = null;
            this.participants = [];
            this.notes = {};
            this.notesWritten = 0;
            this.currentReader = null;
            this.currentNote = null;
            this.notesRemaining = 0;
            this.sessionComplete = false;
            this.receivedNotes = [];
            this.onboardingStep = 'choice';
            this.selectedAction = null;
            this.joinCode = '';
            this.fromDirectLink = false;

            // Clear URL parameters
            const url = new URL(window.location);
            url.search = '';
            window.history.replaceState({}, '', url);

            // Go back to home
            this.currentView = 'home';
        },

        confirmRemoveParticipant(participant) {
            this.participantToRemove = participant;
            document.getElementById('remove_participant_modal').showModal();
        },

        removeParticipant() {
            if (!this.participantToRemove) return;

            this.send({
                type: 'remove_participant',
                data: {
                    participantId: this.participantToRemove.id
                }
            });

            this.participantToRemove = null;
            document.getElementById('remove_participant_modal').close();
        },

        getInitials(name) {
            if (!name) return '?';
            const parts = name.trim().split(/\s+/);
            if (parts.length === 1) {
                return parts[0].substring(0, 2).toUpperCase();
            }
            return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
        },

        getAvatarColor(name) {
            // Generate consistent color based on name
            const colors = [
                'bg-primary text-primary-content',
                'bg-secondary text-secondary-content',
                'bg-accent text-accent-content',
                'bg-info text-info-content',
                'bg-success text-success-content',
                'bg-warning text-warning-content',
            ];

            let hash = 0;
            for (let i = 0; i < name.length; i++) {
                hash = name.charCodeAt(i) + ((hash << 5) - hash);
            }
            return colors[Math.abs(hash) % colors.length];
        },

        showNotification(message, type = 'success') {
            const id = Date.now();
            this.notifications.push({ id, message, type });
            // Auto-dismiss after 4 seconds for errors, 3 for success
            const timeout = type === 'error' ? 4000 : 3000;
            setTimeout(() => {
                this.notifications = this.notifications.filter(n => n.id !== id);
            }, timeout);
        },

        announceToScreenReader(message) {
            this.srAnnouncement = message;
            // Clear after brief moment so same message can be announced again
            setTimeout(() => { this.srAnnouncement = ''; }, 100);
        },

        async copySessionCode() {
            try {
                await navigator.clipboard.writeText(this.sessionCode);
                this.showNotification('Code copied to clipboard!');
            } catch (err) {
                console.error('Failed to copy:', err);
                this.showNotification('Failed to copy code', 'error');
            }
        },

        async copyShareLink() {
            try {
                const shareURL = `${window.location.origin}${window.location.pathname}?code=${this.sessionCode}`;
                await navigator.clipboard.writeText(shareURL);
                this.showNotification('Share link copied to clipboard!');
            } catch (err) {
                console.error('Failed to copy link:', err);
                this.showNotification('Failed to copy link', 'error');
            }
        },

        exportNotesAsText() {
            if (!this.receivedNotes || this.receivedNotes.length === 0) {
                this.showNotification('No notes to export', 'error');
                return;
            }

            const timestamp = new Date().toLocaleString();
            let content = `Uplift - ${this.userName}\n`;
            content += `Session: ${this.sessionCode}\n`;
            content += `Date: ${timestamp}\n`;
            content += `\n${'='.repeat(50)}\n\n`;

            this.receivedNotes.forEach((note, index) => {
                content += `Note ${index + 1} from ${note.authorName}:\n`;
                content += `${note.content}\n\n`;
                content += `${'-'.repeat(50)}\n\n`;
            });

            const blob = new Blob([content], { type: 'text/plain' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `uplift-${this.sessionCode}-${Date.now()}.txt`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);

            this.showNotification('Notes exported as text file!');
        },

        printNotes() {
            if (!this.receivedNotes || this.receivedNotes.length === 0) {
                this.showNotification('No notes to print', 'error');
                return;
            }

            window.print();
        }
    };
}

// Make uplift available globally for Alpine.js x-data
window.uplift = uplift;
