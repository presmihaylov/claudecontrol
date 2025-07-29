# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### ccbackend (Go Backend Server)
```bash
cd ccbackend
make run                    # Run backend server on localhost:3000
make build                  # Build binary to bin/ccbackend
make clean                  # Remove build artifacts
make test                   # Run all tests
make test-verbose           # Run all tests with verbose output
go run cmd/main.go          # Run server directly
go mod tidy                 # Update dependencies
go fmt ./...                # Format Go source files
```

### ccagent (Go CLI Agent)
```bash
cd ccagent
make build                  # Build binary to bin/ccagent
make clean                  # Remove build artifacts
go run cmd/main.go          # Run agent directly
```

### ccfrontend (Next.js Frontend)
```bash
cd ccfrontend
bun dev                     # Run development server on localhost:3000
bun run build               # Build production bundle
bun start                   # Start production server
bun run lint                # Run ESLint
bun install                 # Install dependencies
```

### Supabase (Database Development)
```bash
cd ccbackend
supabase start              # Start local Supabase development environment
supabase stop               # Stop local Supabase services
supabase db reset           # Reset local database with migrations
supabase migration new <name>  # Create new database migration
supabase migration up       # Apply pending migrations
supabase gen types typescript  # Generate TypeScript types from database schema
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
- **ccbackend**: Go HTTP/WebSocket server handling Slack integration with Supabase database
- **ccagent**: Go CLI tool for Claude Code interaction
- **ccfrontend**: Next.js frontend application with React 19 and Tailwind CSS
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
ccbackend requires environment variables (copy `.env.example` to `.env` and configure):
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
- **Supabase**: Database and backend services
- **SQLX**: Enhanced SQL interface (`github.com/jmoiron/sqlx`)
- **Testify**: Testing assertions and utilities (`github.com/stretchr/testify`)
- **PostgreSQL Driver**: Database connectivity (`github.com/lib/pq`)

## Database Architecture

### Schema Management
- **Production Schema**: `claudecontrol` - Contains production tables
- **Test Schema**: `claudecontrol_test` - Isolated test environment
- **Configuration**: Schema name configurable via `DB_SCHEMA` environment variable

### Database Layer (`db/`)
- Uses **sqlx** for enhanced struct scanning without manual field mapping
- Repository pattern with `PostgresAgentsRepository`
- Struct tags: `db:"column_name"` for automatic field mapping
- Error handling: Distinguishes between "not found" and database errors

### Active Agents Table
```sql
CREATE TABLE {schema}.active_agents (
    id UUID PRIMARY KEY,                    -- Auto-generated in service layer
    assigned_job_id UUID NULL,              -- Optional job assignment
    created_at TIMESTAMPTZ DEFAULT NOW(),   -- Auto-managed by database
    updated_at TIMESTAMPTZ DEFAULT NOW()    -- Auto-managed by database
);
```

## Service Layer Patterns

### UUID Management
- **Service-Generated IDs**: Services generate UUIDs internally using `uuid.New()`
- **No External ID Input**: Callers don't provide IDs, ensuring uniqueness
- **Database Timestamps**: `created_at`/`updated_at` managed by database `NOW()`

### Error Handling
- **Validation Errors**: "cannot be nil" for invalid inputs
- **Not Found Errors**: "not found" in error message for missing records
- **Wrapped Errors**: Service layer wraps database errors with context

### Logging Standards
- **Function Entry/Exit Logging**: All service layer functions must include structured logging
- **Starting Log**: `log.Printf("📋 Starting to [action description]")` at function beginning
- **Completion Log**: `log.Printf("📋 Completed successfully - [brief result description]")` before successful return
- **Consistent Emoji**: Use 📋 (clipboard) emoji for both starting and completion logs
- **Descriptive Messages**: Include relevant context like IDs, counts, or operation details

## Testing Standards

### Test Structure
- **Setup/Teardown**: Per-test cleanup using `defer` functions
- **Testify Assertions**: Use `assert.Equal()`, `require.NoError()`, etc.
- **Database Tests**: Use real PostgreSQL test schema, no mocking
- **Error Validation**: Assert specific error message content with `assert.Contains()`

### Test Environment
- **Environment File**: `.env.test` for test-specific configuration
- **Test Schema**: `claudecontrol_test` for isolated testing
- **Cleanup Pattern**: Delete created records in `defer` blocks

## Development Workflow

### After Completing Tasks
**Always build and test to ensure nothing is broken:**
```bash
cd ccbackend && make build  # Build first to catch compilation issues
cd ccbackend && make test   # Then run tests
cd ccfrontend && bun run build && bun run lint  # Build and lint frontend
```

### Database Migrations
- **Apply Pending Migrations**: `supabase migration up` to apply only new migrations
- **Development Reset**: `supabase db reset` to reset and apply all migrations from scratch
- **New Migrations**: `supabase migration new <name>` 
- **Test Schema**: Migrations create both prod and test schemas
```

## Slack Message Handling Guidelines
- **Slack Message Conversion**: Anytime you send a message to slack coming from the ccagent, you should ensure it goes through the `utils.ConvertMarkdownToSlack` function

## Error Handling Guidelines
- **Error Propagation**: Never log errors silently and proceed with control flow. Always propagate the error upstream unless explicitly instructed to log the error and ignore

## Service Layer Architecture Rules
- **User-Scoped Entities**: All entities in the database should be scoped to a user ID. Never manage an entity without a user ID filter for security and data isolation
- **Context-Based User Access**: User ID should be accessed from the context in the service layer instead of being passed explicitly as a parameter
- **Context First Parameter**: All functions in the service layer should take `ctx context.Context` as the first argument to ensure proper request context propagation

## Test Debugging Guidelines
- If you detect that the database is not working when you run tests, you should ask the user to run it for you
```