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

### Dev Mode (Frontend Testing)

The frontend includes a dev mode for testing different session phases without creating real WebSocket sessions. Use URL parameters to specify the phase and number of participants:

**URL Parameters:**
- `dev=true` - Enable dev mode
- `phase=<phase>` - Specify phase: `lobby`, `writing`, `reading`, or `complete` (default: `lobby`)
- `participants=<count>` - Number of fake participants (1-10, default: 3)

**Example URLs:**
```
# Lobby with 5 participants
http://localhost:3001/?dev=true&participants=5

# Writing phase with 4 participants
http://localhost:3001/?dev=true&phase=writing&participants=4

# Reading phase with 3 participants
http://localhost:3001/?dev=true&phase=reading&participants=3

# Complete phase with 5 participants
http://localhost:3001/?dev=true&phase=complete&participants=5
```

### Production Deployment

```bash
npm run build
go build -o uplift ./cmd/server
cp -r dist/* static/
./uplift
```

Set `PORT` environment variable to override default port 8080.

### Issue Tracking (Beads)

This project uses `bd` (beads), a git-backed issue tracker designed for AI agents. Beads provides dependency tracking and ready-work detection across sessions.

**Finding Work:**
```bash
bd ready          # Show issues with no open blockers (start here)
bd blocked        # Show blocked issues
bd stats          # View project statistics
```

**Managing Issues:**
```bash
bd create "Title" -d "Description" -p 1 -t bug
bd list [--status open] [--priority 1]
bd show <issue-id>
bd update <id> --status in_progress
bd close <id> --reason "Completed"
```

**Dependencies:**
```bash
bd dep add <from> <to>                         # Create blocking dependency
bd dep add <from> <to> --type discovered-from  # Link work discovered during task
bd dep tree <issue-id>                         # Visualise dependencies
```

**Dependency Types:**
- `blocks` - Hard blocker (default, affects ready-work detection)
- `discovered-from` - Links new work discovered during execution
- `related` - Soft connection without blocking
- `parent-child` - Epic/subtask hierarchies

**Key Notes:**
- Use `--json` flag on any command for programmatic parsing
- JSONL files are committed to git; changes auto-export with 5-second debounce
- Currently recommended for single-workstream projects only (v0.9.x has multi-workstream bugs)
- Link discovered issues back to parent work using `discovered-from` dependencies

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
- Catppuccin colour themes via @catppuccin/daisyui package
- Four theme variants available:
  - **Latte**: Light theme with warm, pastel colours (default for light mode)
  - **Frappé**: Medium-dark theme with cool, muted colours
  - **Macchiato**: Dark theme with balanced, rich colours
  - **Mocha**: Darkest theme with deep, vibrant colours (default for dark mode)
- Theme switching handled in Alpine.js (`loadTheme()` function) based on system colour scheme preference
- Each Catppuccin theme is imported via separate plugin file in `src/css/catppuccin.*.js`
- Custom animations and print styles in `src/css/styles.css`
