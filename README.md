# Uplift

**[Try it now at upliftapp.online](https://upliftapp.online/)**

A real-time web application for facilitating structured appreciation sessions where team members anonymously write and share positive notes with each other.

## What is Uplift?

Uplift is a guided group activity designed to help teams feel seen and appreciated. Participants join a session, write appreciation notes for their teammates, and then take turns reading these anonymous notes aloud. The anonymity encourages genuine appreciation while the group reading creates a shared moment of connection.

## How It Works

### User Flow

1. **Start or Join**: One person creates a new session and shares the session code (or link) with their team
2. **Lobby**: Participants join using the code and wait in the lobby until everyone arrives
3. **Writing**: The host starts the writing phase. Each participant writes appreciation notes for every other participant
4. **Reading**: Once everyone submits their notes, participants take turns drawing and reading random notes aloud
5. **Completion**: When all notes have been read, participants can export their received notes as text or PDF

### Key Features

- **Real-time synchronisation**: All participants see updates instantly via WebSocket
- **Accessible**: Full keyboard navigation, screen reader support, and ARIA live regions
- **Responsive**: Works on desktop, tablet, and mobile devices
- **Theme support**: Light (Cupcake) and dark (Sunset) themes with system preference detection
- **Session management**: Host controls, participant removal, automatic host reassignment
- **Export options**: Download notes as text file or save as PDF via browser print
- **Auto-reconnect**: Handles network interruptions with exponential backoff
- **Inactivity timeout**: Sessions automatically timeout after 30 minutes of inactivity

## Technical Architecture

### Frontend

- **Alpine.js 3.15.0**: Lightweight reactive framework for UI state management
- **DaisyUI 5.0.0**: Component library built on Tailwind CSS
- **Tailwind CSS 4.0**: Utility-first CSS framework via Vite plugin
- **Vite 7**: Fast build tool and development server

The frontend is a single-page application with no routing. State transitions are handled by Alpine.js and all UI updates happen in response to WebSocket messages or user actions.

### Backend

- **Go 1.25.1**: HTTP server and WebSocket handler
- **Gorilla WebSocket**: WebSocket library for real-time bidirectional communication
- **Standard library**: HTTP server using `net/http`

The backend manages session state, coordinates message passing between clients, and handles WebSocket lifecycle events (connect, disconnect, timeout).

### Communication

The application uses WebSocket for all real-time communication:
- Frontend connects to `/ws` endpoint
- Messages are JSON with `type` and `data` fields
- Backend broadcasts state changes to all session participants
- Automatic reconnection with exponential backoff (1s, 2s, 4s, 8s, 16s, max 30s)

## Prerequisites

- **Go 1.25+**: [Download Go](https://golang.org/dl/)
- **Node.js 18+**: [Download Node.js](https://nodejs.org/)
- **npm**: Comes with Node.js

## Development Setup

### 1. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install Node.js dependencies
npm install
```

### 2. Build the Go Server

```bash
go build -o uplift ./cmd/server
```

### 3. Run in Development Mode

You need two terminal windows running simultaneously:

**Terminal 1 - Go Backend (port 8080):**
```bash
./uplift
```

**Terminal 2 - Vite Dev Server (port 3000):**
```bash
npm run dev
```

The Vite dev server proxies WebSocket connections to the Go backend, so you only need to open `http://localhost:3000` in your browser.

### 4. Build for Production

```bash
# Build frontend assets
npm run build

# Build Go binary
go build -o uplift ./cmd/server

# Copy built assets to static directory
cp -r dist/* static/
```

### 5. Run Production Build

```bash
# Set PORT environment variable (optional, defaults to 8080)
export PORT=8080

# Run the server
./uplift
```

The server will serve static files from `./static` and WebSocket connections on `/ws`.

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── session/
│   │   ├── manager.go        # Session lifecycle management
│   │   └── session.go        # Session state and business logic
│   └── websocket/
│       ├── client.go         # WebSocket client connection handling
│       ├── handler.go        # WebSocket HTTP upgrade handler
│       ├── hub.go            # Central message router
│       └── messagehandler.go # Message processing and session coordination
├── src/
│   ├── css/
│   │   └── styles.css        # Custom styles (animations, print, accessibility)
│   └── js/
│       └── app.js            # Alpine.js application logic
├── static/                   # Production static files (after build)
├── dist/                     # Vite build output (gitignored)
├── index.html                # Main HTML entry point
├── main.js                   # Frontend entry point (imports Alpine, styles)
├── package.json              # Node.js dependencies and scripts
├── go.mod                    # Go module definition
├── go.sum                    # Go dependency checksums
└── vite.config.js            # Vite configuration (Tailwind, proxy, build)
```

## Deployment

### Environment Variables

- `PORT`: HTTP server port (default: `8080`)

### Deployment Steps

1. Build the production assets:
   ```bash
   npm run build
   go build -o uplift ./cmd/server
   cp -r dist/* static/
   ```

2. Deploy the binary and static directory to your hosting platform

3. Ensure the server can accept WebSocket connections (check reverse proxy configuration if using nginx/Apache)

4. Set the `PORT` environment variable if needed

### Platform-Specific Notes

**Heroku:**
- Buildpacks required: `heroku/nodejs` and `heroku/go`
- Procfile: `web: ./uplift`
- PORT is automatically set by Heroku

**Docker:**
```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o uplift ./cmd/server

FROM node:18-alpine AS frontend
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/uplift .
COPY --from=frontend /app/dist ./static
EXPOSE 8080
CMD ["./uplift"]
```

**Fly.io:**
- Use Go buildpack
- Ensure WebSocket support is enabled
- Set internal port to 8080 in fly.toml

## Browser Compatibility

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari, Chrome Mobile)

Requires JavaScript enabled and WebSocket support.

## Licence

MIT
