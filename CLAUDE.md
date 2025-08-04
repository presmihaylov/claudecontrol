# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### ccbackend (Go Backend Server)
```bash
cd ccbackend
make run                    # Run backend server on localhost:8080
make build                  # Build binary to bin/ccbackend
make clean                  # Remove build artifacts
make test                   # Run all tests
make test-verbose           # Run all tests with verbose output
make lint                   # Run golangci-lint checks
make lint-fix               # Run golangci-lint and fix issues automatically
go run cmd/main.go          # Run server directly
go mod tidy                 # Update dependencies
go fmt ./...                # Format Go source files
```

### ccagent (Go CLI Agent)
```bash
cd ccagent
make run                    # Run agent
make build                  # Build binary to bin/ccagent
make clean                  # Remove build artifacts
make build-prod             # Build production binaries for multiple platforms
make lint                   # Run golangci-lint checks
make lint-fix               # Run golangci-lint and fix issues automatically
go run cmd/*.go             # Run agent directly
go run cmd/*.go --bypassPermissions  # Run with bypass permissions (sandbox only)
```

### ccfrontend (Next.js Frontend)
```bash
cd ccfrontend
bun dev                     # Run development server with HTTPS on localhost:3000
bun run dev:http            # Run development server with HTTP (next dev)
bun run build               # Build production bundle
bun start                   # Start production server
bun run lint                # Run Biome linter and check
bun run lint:fix            # Run Biome and fix issues automatically
bun run format              # Format code with Biome
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
go run main.go ws://localhost:8080/ws  # Connect to ccbackend
```

## Architecture Overview

### Multi-Module Structure
- **ccbackend**: Go HTTP/WebSocket server handling Slack integration with Supabase database, Clerk authentication
- **ccagent**: Go CLI tool for Claude Code interaction with WebSocket connection to backend
- **ccfrontend**: Next.js 15 frontend application with React 19, Tailwind CSS 4, and Clerk authentication
- **examples/websockets**: Reference WebSocket implementations (deprecated - see ccagent for current WebSocket patterns)

### Backend Component Organization
- `main.go`: Server setup, environment loading, port 8080
- `handlers/`: HTTP request handlers for Slack events, WebSocket connections, dashboard API
- `middleware/auth.go`: Clerk JWT authentication middleware with user context
- `services/`: Business logic layer with context-based user scoping
- `db/`: Database repository layer using sqlx with PostgreSQL
- `models/`: Domain models and API response structures
- `appctx/`: Application context management for user entities
- `clients/`: External service clients (Slack, WebSocket)
- `usecases/`: Core business use cases and workflows

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
CLERK_SECRET_KEY=<clerk_secret_key>
DB_SCHEMA=<database_schema_name>
DATABASE_URL=<postgresql_connection_string>
```

ccagent requires environment variables:
```
ANTHROPIC_API_KEY=<anthropic_api_key>
CCAGENT_API_KEY=<ccagent_api_key>
CCAGENT_WS_API_URL=<websocket_server_url>  # Optional, defaults to production
```

### Key Integration Points
- **Slack Webhooks**: `/slack/events` endpoint for app mentions and URL verification
- **WebSocket Endpoint**: `/ws` for real-time ccagent connections with API key authentication
- **Dashboard API**: `/api/dashboard/*` endpoints with Clerk JWT authentication
- **Clerk Authentication**: JWT-based user authentication with automatic user creation
- **Claude Code Integration**: ccagent connects via WebSocket with retry logic and job management

## Go Code Standards
- Use `any` instead of `interface{}` for generic types
- WebSocket clients stored as `[]Client` where `Client.ClientConn` is the connection
- Thread-safe operations with `sync.RWMutex` for client management
- JSON message protocol for WebSocket communication

## Module Dependencies

### ccbackend Dependencies
- **Gorilla WebSocket**: Real-time communication (`github.com/gorilla/websocket`)
- **Gorilla Mux**: HTTP routing (`github.com/gorilla/mux`)
- **Slack Go SDK**: Platform integration (`github.com/slack-go/slack`)
- **Clerk SDK**: Authentication (`github.com/clerk/clerk-sdk-go/v2`)
- **SQLX**: Enhanced SQL interface (`github.com/jmoiron/sqlx`)
- **PostgreSQL Driver**: Database connectivity (`github.com/lib/pq`)
- **UUID**: UUID generation (`github.com/google/uuid`)
- **Lo**: Functional utilities (`github.com/samber/lo`)
- **CORS**: Cross-origin requests (`github.com/rs/cors`)
- **Testify**: Testing assertions (`github.com/stretchr/testify`)
- **Godotenv**: Environment management (`github.com/joho/godotenv`)

### ccagent Dependencies
- **Gorilla WebSocket**: WebSocket client (`github.com/gorilla/websocket`)
- **Worker Pool**: Concurrent message processing (`github.com/gammazero/workerpool`)
- **Go Flags**: Command-line argument parsing (`github.com/jessevdk/go-flags`)
- **UUID**: UUID generation (`github.com/google/uuid`)
- **Codename**: Branch name generation (`github.com/lucasepe/codename`)

### ccfrontend Dependencies
- **Next.js 15**: React framework with App Router
- **React 19**: UI library with modern concurrent features
- **Clerk**: Authentication (`@clerk/nextjs`)
- **Tailwind CSS 4**: Utility-first CSS framework
- **Radix UI**: Accessible component primitives
- **Lucide React**: Icon library
- **Biome**: Fast linter and formatter (replaces ESLint/Prettier)

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

### Current Database Schema
The database uses a multi-table design with user-scoped entities:

- **users**: User accounts from Clerk authentication
- **slack_integrations**: Slack workspace integrations per user
- **active_agents**: WebSocket-connected agents per integration
- **jobs**: Tasks assigned to agents
- **agent_job_assignments**: Many-to-many relationship between agents and jobs
- **processed_slack_messages**: Tracks processed Slack messages to avoid duplicates

All core entities are scoped to slack_integration_id for proper user isolation.

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
**Always build, lint, and test to ensure nothing is broken:**
```bash
cd ccbackend && make build  # Build first to catch compilation issues
cd ccbackend && make lint-fix  # Fix linting issues
cd ccbackend && make test   # Then run tests
cd ccagent && make build    # Build ccagent to catch compilation issues
cd ccagent && make lint-fix  # Fix linting issues
cd ccfrontend && bun run build && bun run lint  # Build and lint frontend with Biome
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

## Frontend Architecture

### Authentication & Routing
- **Clerk Integration**: Full authentication with middleware protection
- **Protected Routes**: All routes require authentication by default
- **Next.js App Router**: Modern file-based routing with TypeScript

### Styling & Components
- **Tailwind CSS 4**: Latest version with enhanced features
- **Shadcn/ui Components**: Built on Radix UI primitives
- **Component Library**: Reusable UI components in `src/components/ui/`
- **Biome Configuration**: Strict linting rules with auto-formatting

### Development Tools
- **TypeScript**: Strict type checking throughout
- **Biome**: Fast linting and formatting (replaces ESLint/Prettier)
- **HTTPS Development**: Custom server setup for secure local development

## Agent Architecture (ccagent)

### WebSocket Communication
- **Persistent Connection**: Maintains connection to ccbackend WebSocket server
- **Retry Logic**: Exponential backoff on connection failures
- **Message Types**: Structured message protocol for different operations
- **Worker Pool**: Sequential message processing with job queuing

### Job Management
- **Branch-Based Workflows**: Each job creates/uses a Git branch
- **Pull Request Integration**: Automatic PR creation and management
- **State Persistence**: In-memory job state with branch tracking
- **Idle Job Detection**: Automatic job completion based on PR status

### Claude Integration
- **Session Management**: Persistent Claude Code sessions per job
- **Git Environment**: Automatic Git repository validation and setup
- **Permission Modes**: Support for both `acceptEdits` and `bypassPermissions`
- **Logging**: Comprehensive logging with file output and optional stdout

## Authentication Architecture

### Clerk Integration (ccbackend)
- **JWT Verification**: Middleware validates Clerk JWT tokens
- **User Context**: Automatic user creation and context injection
- **API Protection**: All dashboard endpoints require authentication
- **Error Handling**: Standardized error responses for auth failures

### User Scoping Pattern
- **Context-First**: All service functions take `context.Context` as first parameter
- **User Extraction**: Users extracted from context using `appctx.GetUser()`
- **Database Isolation**: All queries filtered by user's slack_integration_id
- **Security**: No cross-user data access possible