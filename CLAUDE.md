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
make run                    # Run agent (default: Claude)
make build                  # Build binary to bin/ccagent
make clean                  # Remove build artifacts
make build-prod             # Build production binaries for multiple platforms
make lint                   # Run golangci-lint checks
make lint-fix               # Run golangci-lint and fix issues automatically
go run cmd/*.go             # Run agent directly (default: Claude)
go run cmd/*.go --claude-bypass-permissions  # Run with bypass permissions (sandbox only)
go run cmd/*.go --agent claude       # Run with Claude agent (default)
go run cmd/*.go --agent cursor       # Run with Cursor agent
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
bun run typecheck           # Run TypeScript type checking
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

### Examples
```bash
# WebSocket Examples (deprecated)
cd examples/websockets/server
go run main.go             # Start example WebSocket server on :8080

cd examples/websockets/client  
go run main.go             # Connect client to server

# Socket.IO Examples (current)
cd examples/socketio/server
go run main.go             # Start Socket.IO server

cd examples/socketio/client  
go run main.go             # Connect Socket.IO client

# Discord Bot Example
cd examples/discord-bot
go run main.go             # Run Discord bot example

# Worker Pool Example
cd examples/workerpool
go run main.go             # Demonstrate worker pool usage
```

## Architecture Overview

### Multi-Module Structure
- **ccbackend**: Go HTTP/Socket.IO server handling Slack and Discord integrations with Supabase database, Clerk authentication
- **ccagent**: Go CLI tool for Claude Code interaction with Socket.IO connection to backend
- **ccfrontend**: Next.js 15 frontend application with React 19, Tailwind CSS 4, and Clerk authentication
- **examples/websockets**: Reference WebSocket implementations (deprecated - see ccagent for current Socket.IO patterns)
- **examples/socketio**: Socket.IO client/server examples
- **examples/discord-bot**: Discord bot integration example

### Backend Component Organization
- `cmd/main.go`: Server setup, environment loading, port 8080
- `handlers/`: HTTP request handlers for Slack/Discord events, Socket.IO connections, dashboard API
- `middleware/`: Authentication (Clerk JWT) and error alerting middleware
- `services/`: Business logic layer with context-based organization scoping (users, slack_integrations, discord_integrations, jobs, agents)
- `db/`: Database repository layer using sqlx with PostgreSQL
- `models/`: Domain models and API response structures
- `appctx/`: Application context management for organization entities
- `clients/`: External service clients (Slack, Discord, Socket.IO)
- `usecases/`: Core business use cases and workflows (agents, core, discord, slack)
- `config/`: Application configuration management
- `core/`: Core utilities (ULID generation, error handling)
- `testutils/`: Testing utilities and helpers

### Socket.IO Server Architecture
The Socket.IO implementation provides real-time communication:
- Socket.IO server handling agent connections
- Message-based protocol for job management
- Organization-scoped agent tracking
- Integration with Slack and Discord webhooks

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
CCAGENT_API_KEY=<ccagent_api_key>
CCAGENT_SOCKET_URL=<socketio_server_url>  # Optional, defaults to production
```

### Key Integration Points
- **Slack Webhooks**: `/slack/events` endpoint for app mentions and URL verification
- **Discord Webhooks**: `/discord/events` endpoint for Discord message events
- **Socket.IO Endpoint**: Socket.IO server for real-time ccagent connections with API key authentication
- **Dashboard API**: `/api/dashboard/*` endpoints with Clerk JWT authentication
- **Clerk Authentication**: JWT-based user authentication with automatic user creation
- **Claude Code Integration**: ccagent connects via Socket.IO with retry logic and job management

## Go Code Standards
- Use `any` instead of `interface{}` for generic types
- Socket.IO clients managed via organization-scoped connections
- Thread-safe operations with `sync.RWMutex` for client management
- JSON message protocol for Socket.IO communication
- **Prefer `slices.Contains`**: Use `slices.Contains(slice, value)` instead of manual loops when
  checking for membership in a slice
- **ULID IDs**: Use prefixed ULIDs generated via `core.NewID(prefix)` for all entity identifiers

## Module Dependencies

### ccbackend Dependencies
- **Socket.IO**: Real-time communication (`github.com/zishang520/socket.io/v2`)
- **Gorilla Mux**: HTTP routing (`github.com/gorilla/mux`)
- **Slack Go SDK**: Slack integration (`github.com/slack-go/slack`)
- **Discord Go SDK**: Discord integration (`github.com/bwmarrin/discordgo`)
- **Clerk SDK**: Authentication (`github.com/clerk/clerk-sdk-go/v2`)
- **SQLX**: Enhanced SQL interface (`github.com/jmoiron/sqlx`)
- **PostgreSQL Driver**: Database connectivity (`github.com/lib/pq`)
- **ULID**: ULID generation (`github.com/oklog/ulid/v2`)
- **Mo**: Option types (`github.com/samber/mo`)
- **CORS**: Cross-origin requests (`github.com/rs/cors`)
- **Testify**: Testing assertions (`github.com/stretchr/testify`)
- **Godotenv**: Environment management (`github.com/joho/godotenv`)

### ccagent Dependencies
- **Socket.IO Client**: Socket.IO client connection (`github.com/zishang520/socket.io-client-go`)
- **Worker Pool**: Concurrent message processing (`github.com/gammazero/workerpool`)
- **Go Flags**: Command-line argument parsing (`github.com/jessevdk/go-flags`)
- **UUID**: UUID generation (`github.com/google/uuid`)
- **Codename**: Branch name generation (`github.com/lucasepe/codename`)
- **File Lock**: Directory locking (`github.com/gofrs/flock`)

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
- **Option Types**: Uses `mo.Option[*Model]` for Get operations that may return no results
- **Validation Rule**: Repository layer focuses on data access only - **validation should be
  handled at the service layer, not repository layer**
- **Transaction Support**: Transaction management via `txmanager` service for atomic operations
- **Identifier Data Types**: Always use `TEXT` for database identifiers, never `VARCHAR`

### Current Database Schema
The database uses a multi-table design with organization-scoped entities:

- **organizations**: Organizations containing users and integrations
- **users**: User accounts from Clerk authentication, scoped to organizations
- **slack_integrations**: Slack workspace integrations per organization
- **discord_integrations**: Discord guild integrations per organization
- **active_agents**: Socket.IO-connected agents per organization
- **jobs**: Tasks assigned to agents (supports both Slack and Discord)
- **agent_job_assignments**: Many-to-many relationship between agents and jobs
- **processed_slack_messages**: Tracks processed Slack messages to avoid duplicates
- **processed_discord_messages**: Tracks processed Discord messages to avoid duplicates

All core entities are scoped to organization_id for proper data isolation.

## Service Layer Patterns

### ULID Management
- **Service-Generated IDs**: Services generate ULIDs internally using `core.NewID(prefix)`
- **No External ID Input**: Callers don't provide IDs, ensuring uniqueness
- **ULID Format**: Prefixed ULIDs (e.g., `u_01G0EZ1XTM37C5X11SQTDNCTM1` for users)
- **Database Timestamps**: `created_at`/`updated_at` managed by database `NOW()`

### Error Handling
- **Validation Errors**: "cannot be nil" for invalid inputs
- **Not Found Errors**: "not found" in error message for missing records
- **Wrapped Errors**: Service layer wraps database errors with context
- **Option Types**: Uses `mo.Option[*Model]` for Get operations that may return no results

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

### Testing Guidelines
- **Focus on Happy Paths**: Test successful operations and expected business logic
- **Avoid Database Failure Testing**: Do not test database connection failures, transaction rollbacks, or infrastructure failures as these make tests verbose and brittle
- **Test Business Logic**: Focus on service layer validation, data transformations, and business rules
- **Real Database Only**: Use actual PostgreSQL test schema rather than mocking database operations

### Test Execution Notes
- **Flaky Tests**: Tests may occasionally fail due to timing or race conditions - retry running tests 2-3 times before investigating
- **Database Dependency**: Tests require a running PostgreSQL database with proper test schema setup
- **Database Not Started**: If tests fail with database connection errors, abort test fixing and report to user that the database needs to be started with `supabase start`

## Development Workflow

### Testing Approach
- **TestFixer Subagent Available**: Specialized subagent available for automated test execution and fixing
- **Test Commands**: Standard test commands (`make test`, `go test`, `bun test`) are available for each module
- **Test Environment**: Isolated test database schema for safe testing

### Build Verification
- **Backend Build**: `cd ccbackend && make build`
- **Agent Build**: `cd ccagent && make build`  
- **Frontend Build**: `cd ccfrontend && bun run build`

### Code Quality Tools
- **Linting Available**: Each module has lint commands (`make lint`, `bun run lint`)
- **Formatting Available**: Automated formatting tools for consistent code style
- **Auto-fix Capabilities**: Lint-fix commands available for automated corrections

### Database Migrations
- **Apply Pending Migrations**: `supabase migration up` to apply only new migrations
- **Development Reset**: `supabase db reset` to reset and apply all migrations from scratch
- **New Migrations**: `supabase migration new <name>` 
- **Migration File Creation**: ALWAYS use `supabase migration new <name>` to generate migration files instead of manually determining timestamps
- **Test Schema**: Migrations create both prod and test schemas
```

## Message Handling Guidelines
- **Slack Message Conversion**: Anytime you send a message to slack coming from the ccagent,
  you should ensure it goes through the `utils.ConvertMarkdownToSlack` function
- **Discord Message Support**: Discord integration supports message events and responses
- **Multi-Platform Support**: Jobs can be created from both Slack and Discord interactions

## Error Handling Guidelines
- **Error Propagation**: Never log errors silently and proceed with control flow. Always
  propagate the error upstream unless explicitly instructed to log the error and ignore
- **No Silent Failures**: Avoid patterns like `if err != nil { log.Printf(...); }` without
  returning the error. This hides failures and makes debugging difficult
- **Proper Error Wrapping**: Use `fmt.Errorf("context: %w", err)` to wrap errors with context
  when propagating upstream
- **Critical Operations**: For critical operations (database writes, external API calls, job
  cleanup), always return errors to the caller
- **Log and Return**: When an error occurs, log it for debugging AND return it for proper
  handling: `log.Printf(...); return fmt.Errorf(...)`

## Service Layer Architecture Rules
- **Organization-Scoped Entities**: All entities in the database should be scoped to an organization ID.
  Never manage an entity without organization ID filter for security and data isolation
- **Context-Based Organization Access**: Organization should be accessed from the context in the service
  layer instead of being passed explicitly as a parameter
- **Context First Parameter**: All functions in the service layer should take
  `ctx context.Context` as the first argument to ensure proper request context propagation

## Service Architecture Pattern

The codebase follows a standardized service architecture pattern for maintainability and
consistency. **All new services must follow this pattern.**

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
// Validation errors - no whitespace between validations
if param == "" {
    return nil, fmt.Errorf("param cannot be empty")
}
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
- **Layer Separation**: Service layer handles all validation (ULID format, required fields,
  business rules) - repository layer focuses purely on database operations

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

### Socket.IO Communication
- **Persistent Connection**: Maintains connection to ccbackend Socket.IO server
- **Retry Logic**: Exponential backoff on connection failures
- **Message Types**: Structured message protocol for different operations
- **Worker Pool**: Sequential message processing with job queuing

### Job Management
- **Branch-Based Workflows**: Each job creates/uses a Git branch
- **Pull Request Integration**: Automatic PR creation and management
- **State Persistence**: In-memory job state with branch tracking
- **Idle Job Detection**: Automatic job completion based on PR status

### CLI Agent Integration
- **Multiple Agents**: Support for both Claude Code (`--agent claude`) and Cursor (`--agent cursor`)
- **Session Management**: Persistent agent sessions per job with automatic resumption
- **Git Environment**: Automatic Git repository validation and setup
- **Permission Modes**: Support for both `acceptEdits` and `bypassPermissions` (Claude only)
- **Logging**: Comprehensive logging with agent-specific file output and optional stdout
- **Default Agent**: Claude Code is the default agent when no `--agent` flag is specified

## Authentication Architecture

### Clerk Integration (ccbackend)
- **JWT Verification**: Middleware validates Clerk JWT tokens
- **User Context**: Automatic user creation and context injection
- **API Protection**: All dashboard endpoints require authentication
- **Error Handling**: Standardized error responses for auth failures

### Organization Scoping Pattern
- **Context-First**: All service functions take `context.Context` as first parameter
- **Organization Extraction**: Organizations extracted from context using `appctx.GetOrganization()`
- **Database Isolation**: All queries filtered by organization_id
- **Security**: No cross-organization data access possible

## Available Subagents

Claude Code has access to specialized subagents for specific tasks. These should be used proactively when appropriate:

### Researcher Subagent
- **Purpose**: Research APIs, libraries, SDKs, and development tools using Context7 documentation and web search. Provides comprehensive analysis of available functionality and implementation guidance for specific tools and libraries.
- **When to Use**: 
  - Before implementing any new library, API, or SDK integration
  - When encountering unfamiliar technologies or tools in the codebase
  - To evaluate alternative solutions for technical requirements
  - When investigating compatibility, performance, or security considerations
  - To understand best practices for specific technology stacks
  - Before making architectural decisions that involve external dependencies
- **Capabilities**: 
  - **Context7 Documentation Access**: Retrieves up-to-date official documentation and code examples
  - **Web Search Integration**: Finds latest tutorials, best practices, and community resources
  - **Implementation Guidance**: Provides concrete examples and integration patterns
  - **Feature Analysis**: Determines availability of specific functionality and limitations
  - **Alternative Evaluation**: Compares different approaches and recommends optimal solutions
  - **Version Compatibility**: Identifies version requirements and compatibility constraints
  - **Security Assessment**: Highlights security considerations and best practices
- **Usage Examples**:
  - **New Integration**: "Research how to integrate Discord webhooks with Go backend"
  - **Library Evaluation**: "Compare WebSocket libraries for real-time communication in Go"
  - **API Investigation**: "Research Slack API rate limits and authentication patterns"
  - **Technology Assessment**: "Evaluate PostgreSQL vs MongoDB for our data storage needs"
  - **Best Practices**: "Research Go error handling patterns for web services"
- **Expected Output**: Structured research reports with implementation guidance, code examples, alternatives analysis, and actionable recommendations
- **Proactive Usage**: Use the researcher subagent immediately when you encounter any unfamiliar technology rather than attempting implementation without proper research

### Code Reviewer Subagent  
- **Purpose**: Comprehensive code review of all changes on current branch vs main branch
- **Capabilities**: Reviews Go backend, CLI agent, and Next.js frontend code for bugs, performance, security, and adherence to project conventions
- **Usage**: Available for detailed code analysis when needed

### TestFixer Subagent
- **Purpose**: Automatically fixes broken tests and suggests new test coverage
- **Prerequisites**: **MUST check if Supabase is running before executing any tests**
  - Run `supabase status` or equivalent to verify database availability
  - If Supabase is not running, immediately stop execution and instruct user to run `supabase start`
  - All tests will fail without a running database, making test fixing impossible
- **Capabilities**: Fixes Go tests in ccbackend and ccagent, suggests new test coverage, runs test suites autonomously
- **Usage**: Available for automated test management and fixing test issues, but only after confirming database availability

## Mock Architecture

### Mock Library and Location
- **Library**: Uses **Testify Mock** (`github.com/stretchr/testify/mock`) for all mocking
- **Co-located Mocks**: Each service mock is defined alongside its implementation in `services/{servicename}/{servicename}_mock.go`
- **Manual Mock Creation**: No code generation - all mocks are manually implemented
- **Package-Level Mocks**: Mocks belong to the same package as the service they mock

### Mock Naming Convention
- **Pattern**: `Mock{ServiceName}Service` (e.g., `MockUsersService`, `MockSlackIntegrationsService`)
- **Interface Implementation**: All mocks implement their corresponding service interfaces from `services/services.go`
- **Struct Embedding**: Each mock embeds `mock.Mock` from testify

### Mock Implementation Pattern

#### 1. Define Mock Struct
```go
// In services/yourservice/yourservice_mock.go
package yourservice

type MockYourService struct {
    mock.Mock
}
```

#### 2. Implement Interface Methods
```go
func (m *MockYourService) CreateEntity(ctx context.Context, param1 string, userID string) (*models.Entity, error) {
    args := m.Called(ctx, param1, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.Entity), args.Error(1)
}

func (m *MockYourService) GetEntityByID(ctx context.Context, id string) (mo.Option[*models.Entity], error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return mo.None[*models.Entity](), args.Error(1)
    }
    return args.Get(0).(mo.Option[*models.Entity]), args.Error(1)
}
```

#### 3. Use Mocks in Tests
```go
// In handler test files
func TestSomeHandler(t *testing.T) {
    mockUsersService := &MockUsersService{}
    mockSlackService := &MockSlackIntegrationsService{}
    
    // Setup expectations
    mockUsersService.On("GetOrCreateUser", mock.Anything, "clerk", "user_123").
        Return(testUser, nil)
    
    // Create handler with mocks
    handler := &DashboardHandler{
        usersService: mockUsersService,
        slackIntegrationsService: mockSlackService,
    }
    
    // Run test...
    
    // Assert expectations were met
    mockUsersService.AssertExpectations(t)
}
```

### Mock Return Value Patterns

#### Standard Returns
```go
// For pointer returns - check for nil
if args.Get(0) == nil {
    return nil, args.Error(1)
}
return args.Get(0).(*models.Entity), args.Error(1)
```

#### Option Type Returns (mo.Option)
```go
// For Option types - return None() when nil
if args.Get(0) == nil {
    return mo.None[*models.Entity](), args.Error(1)
}
return args.Get(0).(mo.Option[*models.Entity]), args.Error(1)
```

#### Slice Returns
```go
// For slice returns
if args.Get(0) == nil {
    return nil, args.Error(1)
}
return args.Get(0).([]*models.Entity), args.Error(1)
```

#### Error-Only Returns
```go
// For methods that only return error
args := m.Called(ctx, param1, param2)
return args.Error(0)
```

### Mock Organization Rules

#### File Location
- **Service mocks**: Co-located in `services/{servicename}/{servicename}_mock.go`
- **Handler mocks**: Legacy mocks in `handlers/dashboard_mocks.go` (being phased out)
- **Repository mocks**: Create in `db/` if needed (rare - prefer real database in tests)
- **External client mocks**: Create service-specific files if extensive

#### Import Requirements
```go
import (
    "context"
    "github.com/samber/mo"              // For Option types
    "github.com/stretchr/testify/mock"  // For mock.Mock
    "ccbackend/models"                  // For domain models
)
```

#### Testing Best Practices
- **Real Database for Service Tests**: Use actual PostgreSQL test schema, not repository mocks
- **Handler Tests Use Mocks**: HTTP handlers mock their service dependencies
- **Mock Only External Boundaries**: Mock services, not internal data structures
- **Assert Expectations**: Always call `mock.AssertExpectations(t)` in tests

### Creating New Service Mocks

When adding a new service, create its mock alongside the implementation:

1. **Create Mock File**: `services/{servicename}/{servicename}_mock.go`
2. **Add Mock Struct**: `type MockServiceName struct { mock.Mock }`
3. **Implement All Interface Methods**: Match the service interface exactly from `services/services.go`
4. **Follow Return Patterns**: Use appropriate null-checking for different return types
5. **Same Package**: Mock must be in the same package as the service for proper access

#### Example Structure
```
services/
├── servicename/
│   ├── servicename.go      # Service implementation  
│   ├── servicename_mock.go # Mock implementation
│   └── servicename_test.go # Service tests
```

This co-located approach keeps mocks close to their implementations and makes them easy to find and maintain.

## Linear Workflow Guidelines

### Ticket Status Management
- **Never Move to Done**: NEVER move Linear tickets to "Done" status unless explicitly instructed by the user
- **In Progress Movement**: When starting work on a ticket, move it to "In Progress" status  
- **Status Updates**: Only move tickets between statuses when explicitly requested or when beginning work
- **User Authorization Required**: All ticket completion must be user-authorized - Claude Control cannot determine when work is truly complete from the user's perspective
