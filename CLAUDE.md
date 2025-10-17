# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Uplift is a real-time web application for facilitating structured appreciation sessions where colleagues write and share positive notes about each other. The architecture is a hybrid Go backend + Alpine.js frontend connected via WebSockets.

## Development Commands

### Building

```bash
# Build Go server
go build -o uplift ./cmd/server

# Build frontend assets
npm run build
```

### Running Development Environment

Development requires two concurrent processes:

```bash
# Terminal 1: Run Go backend (port 8080)
./uplift

# Terminal 2: Run Vite dev server (port 3000)
npm run dev
```

The Vite dev server proxies WebSocket connections (`/ws`) to the Go backend. Access the application at `http://localhost:3000`.

### Testing

Currently, there are no automated tests in this codebase. When implementing tests:
- Follow TDD practices
- Create unit tests for Go packages under `internal/`
- Create frontend tests for Alpine.js components
- Add integration tests for WebSocket message handling

### Production Deployment

```bash
npm run build
go build -o uplift ./cmd/server
cp -r dist/* static/
./uplift
```

Set `PORT` environment variable to override default port 8080.

## Architecture

### Backend (Go)

The Go backend uses a hub-and-spoke WebSocket architecture:

- **Hub** (`internal/websocket/hub.go`): Central message router managing all WebSocket connections. Uses channels for registration, unregistration, and message processing. All client connections are organised by session ID.

- **MessageHandler** (`internal/websocket/messagehandler.go`): Processes incoming WebSocket messages and coordinates with SessionManager. Handles business logic for all message types (create session, join session, start writing, submit notes, draw note, etc.).

- **SessionManager** (`internal/session/manager.go`): Thread-safe in-memory storage for active sessions. Provides lookup by session ID or session code. Sessions are ephemeral and exist only in memory.

- **Session** (`internal/session/session.go`): Contains session state (phase, participants, notes, etc.) with mutex-protected state transitions. Handles all session business logic including host reassignment, note shuffling, and phase progression.

### Frontend (Alpine.js)

Single-page application with no routing. All state is managed in one Alpine.js component (`src/js/app.js`):

- State transitions happen via `currentView` property (home → create/join → lobby → writing → reading → complete)
- All UI updates are reactive to WebSocket messages or local state changes
- WebSocket reconnection uses exponential backoff (1s, 2s, 4s, 8s, 16s, max 30s)

### WebSocket Message Protocol

All messages are JSON with `type` and `data` fields:

```go
type Message struct {
    Type string      `json:"type"`
    Data interface{} `json:"data"`
}
```

Critical message types:
- `create_session`, `join_session`: Session lifecycle
- `start_writing`: Transition from lobby to writing phase
- `submit_notes`: Submit appreciation notes for all participants
- `draw_note`: Request next random note during reading phase
- `state_update`: Server broadcasts session state changes to all clients

### State Synchronisation

The backend is source of truth for all session state. When state changes:
1. Session updates its internal state (with mutex lock)
2. MessageHandler broadcasts `state_update` to all session participants via Hub
3. All connected clients receive the update and synchronise their local state

### Go Module Structure

The `go.mod` declares this as `github.com/cassiascheffer/uplift`. All internal imports use this module path.

## Key Architectural Constraints

- **No database**: All session data is in-memory only. Sessions are lost on server restart.
- **No authentication**: Users are identified by randomly generated IDs and self-declared names.
- **Ephemeral sessions**: Sessions timeout after 30 minutes of inactivity (handled in Session struct).
- **Single-page application**: No client-side routing, all navigation is view state changes.

## Styling

- Uses Tailwind CSS 4.0 via Vite plugin
- DaisyUI 5.0 provides component classes
- Two themes: "cupcake" (light) and "sunset" (dark)
- Theme switching handled in Alpine.js with localStorage persistence
- Custom animations and print styles in `src/css/styles.css`
