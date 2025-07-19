# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### ccbackend (Go Backend Server)
```bash
cd ccbackend
go run .                    # Run backend server on localhost:3000
go mod tidy                 # Update dependencies
```

### ccagent (Go CLI Agent)
```bash
cd ccagent
make build                  # Build binary to bin/ccagent
make clean                  # Remove build artifacts
go run cmd/main.go          # Run agent directly
```

### ccagent-ts (TypeScript CLI)
```bash
cd ccagent-ts
bun install                 # Install dependencies
bun run bin:build          # Build CLI to dist/ccagent.js
bun test                   # Run tests
bun run version:patch      # Bump version and prepare for publish
```

### WebSocket Examples
```bash
cd examples/websockets/server
go run main.go             # Start example server on :8080

cd examples/websockets/client  
go run main.go             # Connect client to server
go run main.go ws://localhost:3000/ws  # Connect to ccbackend
```

## Architecture Overview

### Multi-Module Structure
- **ccbackend**: Go HTTP/WebSocket server handling Slack integration
- **ccagent**: Go CLI tool for Claude Code interaction
- **ccagent-ts**: TypeScript CLI tool (NPM package `@presmihaylov/ccagent`)
- **examples/websockets**: Reference WebSocket implementations

### Backend Component Organization
- `main.go`: Server setup, environment loading, port 3000
- `commands.go`: Slack slash command handlers (`/cc` command)
- `events.go`: Slack event processing (app mentions, URL verification)  
- `websockets.go`: WebSocket server with pluggable message handlers

### WebSocket Server Architecture
The WebSocket implementation uses a refactored design:
- `newWebsocketServer()`: Creates server instance
- `startWebsocketServer()`: Registers HTTP handlers
- `getClients()`: Returns connected clients array
- `sendMessage(client, msg)`: Send arbitrary message to client
- `registerMessageHandler(func)`: Register message processors

All messages received from clients invoke all registered handlers, enabling modular message processing.

### Environment Configuration
ccbackend requires environment variables (use `.env` file):
```
SLACK_SIGNING_SECRET=<slack_signing_secret>
SLACK_BOT_TOKEN=<slack_bot_token>
```

### Key Integration Points
- **Slack Webhooks**: `/slack/commands` and `/slack/events` endpoints
- **WebSocket Endpoint**: `/ws` for real-time client connections
- **Claude Code Integration**: ccagent executes with `--permission-mode bypassPermissions`

## Go Code Standards
- Use `any` instead of `interface{}` for generic types
- WebSocket clients stored as `[]Client` where `Client.ClientConn` is the connection
- Thread-safe operations with `sync.RWMutex` for client management
- JSON message protocol for WebSocket communication

## Module Dependencies
- **Gorilla WebSocket**: Real-time communication (`github.com/gorilla/websocket`)
- **Slack Go SDK**: Platform integration (`github.com/slack-go/slack`)
- **Godotenv**: Environment management (`github.com/joho/godotenv`)
- **Bun**: TypeScript runtime and build tool