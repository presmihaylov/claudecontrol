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
- **No Silent Failures**: Avoid patterns like `if err != nil { log.Printf(...); }` without returning the error. This hides failures and makes debugging difficult
- **Proper Error Wrapping**: Use `fmt.Errorf("context: %w", err)` to wrap errors with context when propagating upstream
- **Critical Operations**: For critical operations (database writes, external API calls, job cleanup), always return errors to the caller
- **Log and Return**: When an error occurs, log it for debugging AND return it for proper handling: `log.Printf(...); return fmt.Errorf(...)`

## Service Layer Architecture Rules
- **User-Scoped Entities**: All entities in the database should be scoped to a user ID. Never manage an entity without a user ID filter for security and data isolation
- **Context-Based User Access**: User ID should be accessed from the context in the service layer instead of being passed explicitly as a parameter
- **Context First Parameter**: All functions in the service layer should take `ctx context.Context` as the first argument to ensure proper request context propagation

## Service Architecture Pattern

The codebase follows a standardized service architecture pattern for maintainability and consistency. **All new services must follow this pattern.**

### Service Organization Structure
```
services/
├── services.go              # Interface definitions only
├── servicename/            # Individual service packages
│   ├── servicename.go      # Service implementation
│   └── servicename_test.go # Service tests
└── anothersvc/
    ├── anothersvc.go
    └── anothersvc_test.go
```

### Interface-First Design
1. **Define Interface in `services/services.go`**:
```go
// ExampleService defines the interface for example operations
type ExampleService interface {
    CreateExample(ctx context.Context, name string, userID string) (*models.Example, error)
    GetExamplesByUserID(ctx context.Context, userID string) ([]*models.Example, error)
    GetExampleByID(ctx context.Context, id string) (*models.Example, error)
    UpdateExample(ctx context.Context, id string, updates map[string]any) (*models.Example, error)
    DeleteExample(ctx context.Context, id string) error
}
```

2. **Implement in Dedicated Package** (`services/examples/examples.go`):
```go
package examples

import (
    "context"
    "fmt"
    "log"

    "ccbackend/core"
    "ccbackend/db"
    "ccbackend/models"
)

type ExamplesService struct {
    examplesRepo *db.PostgresExamplesRepository
}

func NewExamplesService(repo *db.PostgresExamplesRepository) *ExamplesService {
    return &ExamplesService{examplesRepo: repo}
}

func (s *ExamplesService) CreateExample(ctx context.Context, name string, userID string) (*models.Example, error) {
    log.Printf("📋 Starting to create example: %s for user: %s", name, userID)
    
    if name == "" {
        return nil, fmt.Errorf("name cannot be empty")
    }
    if !core.IsValidULID(userID) {
        return nil, fmt.Errorf("user ID must be a valid ULID")
    }

    example := &models.Example{
        ID:     core.NewID("ex"),
        Name:   name,
        UserID: userID,
    }

    if err := s.examplesRepo.CreateExample(ctx, example); err != nil {
        return nil, fmt.Errorf("failed to create example: %w", err)
    }

    log.Printf("📋 Completed successfully - created example with ID: %s", example.ID)
    return example, nil
}
// ... implement other interface methods
```

### Creating New Services - Step by Step

#### 1. Define the Interface
Add your service interface to `services/services.go`:
```go
type YourNewService interface {
    // Follow naming pattern: action + entity + context signature
    CreateEntity(ctx context.Context, param1 string, userID string) (*models.Entity, error)
    GetEntitiesByUserID(ctx context.Context, userID string) ([]*models.Entity, error)
    GetEntityByID(ctx context.Context, id string) (*models.Entity, error)
    UpdateEntity(ctx context.Context, id string, updates map[string]any) (*models.Entity, error)
    DeleteEntity(ctx context.Context, id string) error
}
```

#### 2. Create Service Package
```bash
mkdir services/yournewservice
touch services/yournewservice/yournewservice.go
touch services/yournewservice/yournewservice_test.go
```

#### 3. Implement Service
In `services/yournewservice/yournewservice.go`:
```go
package yournewservice

// Follow exact pattern from existing services
type YourNewServiceImpl struct {
    repo *db.PostgresYourEntityRepository
}

func NewYourNewService(repo *db.PostgresYourEntityRepository) *YourNewServiceImpl {
    return &YourNewServiceImpl{repo: repo}
}
```

#### 4. Update Database Layer
Create corresponding repository in `db/yourentity.go` following the database patterns.

#### 5. Wire Up in Main
In `cmd/main.go`, add initialization:
```go
import yournewservice "ccbackend/services/yournewservice"

// In main function:
yourRepo := db.NewPostgresYourEntityRepository(dbConn, cfg.DatabaseSchema)
yourService := yournewservice.NewYourNewService(yourRepo)
```

#### 6. Create Mock for Tests
In `handlers/dashboard_mocks.go` (or create service-specific mock):
```go
type MockYourNewService struct {
    mock.Mock
}

func (m *MockYourNewService) CreateEntity(ctx context.Context, param1 string, userID string) (*models.Entity, error) {
    args := m.Called(ctx, param1, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.Entity), args.Error(1)
}
// ... implement other mock methods
```

### Mandatory Service Conventions

#### Function Signatures
- **Always** `ctx context.Context` as first parameter
- **Consistent naming**: `CreateX`, `GetXByY`, `UpdateX`, `DeleteX`  
- **User scoping**: Include `userID string` parameter for user-scoped entities
- **Return patterns**: `(*models.Entity, error)` or `([]*models.Entity, error)` or `error`

#### Error Handling
```go
// Validation errors
if param == "" {
    return nil, fmt.Errorf("param cannot be empty")
}

// ULID validation
if !core.IsValidULID(userID) {
    return nil, fmt.Errorf("user ID must be a valid ULID")
}

// Database errors
if err := s.repo.SomeOperation(ctx, ...); err != nil {
    return nil, fmt.Errorf("failed to perform operation: %w", err)
}
```

#### Logging Pattern
```go
func (s *Service) SomeOperation(ctx context.Context, param string) (*models.Entity, error) {
    log.Printf("📋 Starting to [action description]: %s", param)
    
    // ... implementation ...
    
    log.Printf("📋 Completed successfully - [result description]: %s", result.ID)
    return result, nil
}
```

#### Database Integration
- **All repository calls** must pass `ctx` as first parameter
- **Use context-aware sqlx functions**: `QueryRowxContext`, `SelectContext`, `GetContext`, `ExecContext`
- **Repository pattern**: Create matching repository in `db/` package

### Package Naming Rules
- **Package names**: Use singular, lowercase, no underscores (e.g., `examples`, `users`, not `user_profiles`)
- **File names**: Match package name (e.g., `examples.go` in `examples/` package)
- **Import aliases**: Use descriptive aliases when needed (`import examples "ccbackend/services/examples"`)

### Testing Requirements
- **Test file**: `servicename_test.go` in same package as service
- **Real database**: Use PostgreSQL test schema, no mocking for database tests
- **Context usage**: Pass `context.Background()` or request context in tests
- **Cleanup**: Use `defer` functions to clean up test data

This pattern ensures consistency, maintainability, and proper separation of concerns across all services.

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