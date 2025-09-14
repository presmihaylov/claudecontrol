# ccbackend

Go-based backend server for Claude Control, providing real-time communication with AI agents through Socket.IO and integrations with Slack and Discord.

## Overview

ccbackend is a comprehensive HTTP server that handles:
- **Socket.IO Communication**: Real-time bidirectional communication with ccagent clients
- **Slack Integration**: Webhook handling for Slack app mentions and events
- **Discord Integration**: Webhook handling for Discord bot interactions
- **Authentication**: Clerk JWT-based user authentication
- **Database Layer**: PostgreSQL with organization-scoped data isolation
- **Job Management**: Task assignment and execution tracking for AI agents

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database
- Supabase CLI (for database migrations)
- make

## Environment Configuration

Create a `.env` file in the ccbackend directory with the following variables:

```env
# Slack Integration
SLACK_SIGNING_SECRET=your_slack_signing_secret
SLACK_CLIENT_ID=your_slack_client_id
SLACK_CLIENT_SECRET=your_slack_client_secret
DISCORD_BOT_TOKEN=your_discord_bot_token

# Discord Integration
DISCORD_CLIENT_ID=your_discord_client_id
DISCORD_CLIENT_SECRET=your_discord_client_secret

# GitHub Integration
GITHUB_APP_ID=your_github_app_id
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_APP_PRIVATE_KEY=your_github_app_private_key_base64

# Clerk Authentication
CLERK_SECRET_KEY=sk_test_your_clerk_secret_key

# Database Configuration
DB_URL=postgresql://username:password@localhost:5432/database_name
DB_SCHEMA=claudecontrol

# Server Configuration
PORT=8080
ENVIRONMENT=development
CORS_ALLOWED_ORIGINS=https://localhost:3000,https://yourdomain.com

# SSH Configuration (for agent operations)
DEFAULT_SSH_HOST=your_ssh_host
SSH_PRIVATE_KEY_B64=your_base64_encoded_ssh_private_key
```

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `SLACK_SIGNING_SECRET` | Slack app signing secret for webhook verification | No |
| `SLACK_CLIENT_ID` | Slack OAuth client ID | No |
| `SLACK_CLIENT_SECRET` | Slack OAuth client secret | No |
| `DISCORD_BOT_TOKEN` | Discord bot token for bot operations | No |
| `DISCORD_CLIENT_ID` | Discord OAuth client ID | No |
| `DISCORD_CLIENT_SECRET` | Discord OAuth client secret | No |
| `GITHUB_APP_ID` | GitHub App ID for GitHub integration | No |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID | No |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth client secret | No |
| `GITHUB_APP_PRIVATE_KEY` | Base64-encoded GitHub App private key | No |
| `CLERK_SECRET_KEY` | Clerk secret key for JWT verification | No |
| `DB_URL` | PostgreSQL connection string | Yes |
| `DB_SCHEMA` | Database schema name (default: `claudecontrol`) | Yes |
| `PORT` | Server port (default: 8080) | No |
| `ENVIRONMENT` | Deployment environment (development/production) | No |
| `CORS_ALLOWED_ORIGINS` | Comma-separated list of allowed CORS origins | No |
| `DEFAULT_SSH_HOST` | Default SSH host for deploying ccagents | No |
| `SSH_PRIVATE_KEY_B64` | Base64-encoded SSH private key for agent operations | No |

## Database Setup

The backend uses PostgreSQL with Supabase migrations for schema management.

### Initialize Database

```bash
# Apply all database migrations
./scripts/ccdbup.sh

# To reset database (drops all schemas)
./scripts/ccdbdown.sh
```

## Development Commands
```bash
# Run development server (localhost:8080)
make run

# Build binary
make build

# Run all tests (against local DB)
make test
```

## API Endpoints

### Health Check
- `GET /health` - Server health status

### Slack Integration
- `POST /slack/events` - Slack webhook events (app mentions, URL verification)

### Discord Integration
- `POST /discord/events` - Discord webhook events (message events)

### Dashboard API
- `GET /api/dashboard/*` - Protected dashboard endpoints (requires Clerk JWT)

### Socket.IO
- Socket.IO server on same port for real-time ccagent communication
- API key-based authentication for agents

