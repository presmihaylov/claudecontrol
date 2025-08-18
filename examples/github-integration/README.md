# GitHub Integration Example

This example demonstrates how to integrate with GitHub using OAuth to authenticate and get access to repositories.

## What This Example Does

1. **OAuth Flow**: Implements GitHub OAuth 2.0 flow to authenticate users
2. **Repository Selection**: Allows users to choose which repositories to grant access to
3. **Token Generation**: Generates an access token for API calls
4. **Usage Instructions**: Provides clear instructions on how to use the generated token

## Setup Instructions

### 1. Create a GitHub App

1. Go to [GitHub Apps settings](https://github.com/settings/apps)
2. Click "New GitHub App"
3. Fill in the required fields:
   - **GitHub App name**: "Your App Name"
   - **Homepage URL**: `http://localhost:8080`
   - **Authorization callback URL**: `http://localhost:8080/callback`
4. Set permissions:
   - **Repository permissions > Contents**: Read
   - **Repository permissions > Metadata**: Read
   - **Account permissions > Email addresses**: Read (optional)
5. Click "Create GitHub App"
6. Copy the **Client ID** and generate a **Client Secret**

### 2. Configure Environment

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env` and replace the placeholder values:
   ```
   GITHUB_CLIENT_ID=your_actual_client_id
   GITHUB_CLIENT_SECRET=your_actual_client_secret
   ```

### 3. Run the Example

```bash
# Install dependencies
go mod tidy

# Run the example
go run main.go
```

## Usage Flow

1. **Start Server**: Run `go run main.go`
2. **Open Browser**: Navigate to `http://localhost:8080`
3. **Click Connect**: Click the "Connect to GitHub" button
4. **Authorize App**: You'll be redirected to GitHub to authorize the app
5. **Select Repositories**: Choose which repositories to grant access to
6. **Get Token**: You'll be redirected back with your access token
7. **Use Token**: Follow the provided instructions to use the token

## What You Get

After completing the OAuth flow, you'll receive:

- **Access Token**: A token to authenticate API requests
- **Repository List**: List of repositories you have access to
- **Usage Examples**: How to use the token with:
  - GitHub API calls
  - Git commands
  - Application integration

## Example Usage of Generated Token

### With GitHub API
```bash
curl -H "Authorization: token YOUR_TOKEN" \
     -H "Accept: application/vnd.github.v3+json" \
     https://api.github.com/user/repos
```

### With Git Commands
```bash
git clone https://YOUR_TOKEN@github.com/username/repository.git
```

### In Applications
```bash
# Set as environment variable
export GITHUB_TOKEN="YOUR_TOKEN"

# Use in HTTP headers
Authorization: token YOUR_TOKEN
```

## Security Notes

- ⚠️ **Keep your token secure** - never commit it to version control
- 🔐 **Use environment variables** to store tokens in production
- 🔄 **Tokens can be revoked** anytime in your GitHub settings
- 🎯 **Scoped access** - tokens only work with repositories you selected

## Architecture

The example includes:

- **HTTP Server**: Serves the OAuth flow pages
- **OAuth Implementation**: Handles the complete GitHub OAuth 2.0 flow
- **Token Exchange**: Exchanges authorization code for access token
- **API Integration**: Makes authenticated requests to GitHub API
- **User Interface**: Simple HTML interface for the OAuth flow

## Extending This Example

You can extend this example to:

- Store tokens in a database
- Implement token refresh
- Add more GitHub API integrations
- Build a full repository management interface
- Integrate with webhooks for real-time updates