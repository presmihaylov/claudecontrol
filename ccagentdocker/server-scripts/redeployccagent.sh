#!/bin/bash

# Script to redeploy ccagent services with token-rotator to docker-compose.yml
# This script performs an upsert - adds new services or updates existing ones

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Required options:"
    echo "  -n, --name NAME                    Instance name (e.g., 'test', 'prod')"
    echo "  -k, --api-key KEY                  CCAGENT_API_KEY"
    echo "  -r, --repo-url URL                 Repository URL (e.g., 'github.com/user/repo')"
    echo "  -i, --installation-id ID           GitHub Installation ID"
    echo ""
    echo "Authentication (choose one):"
    echo "  -a, --anthropic-key KEY            Anthropic API Key"
    echo "  -o, --oauth-token TOKEN            Claude Code OAuth Token"
    echo ""
    echo "Optional:"
    echo "  -f, --file FILE                    Docker compose file (default: docker-compose.yml)"
    echo "  -c, --config-only                  Only update docker compose config, don't redeploy services"
    echo ""
    echo "Example:"
    echo "  $0 -n test -k sys_xxx -r github.com/user/repo -i 12345 -o sk-ant-xxx"
    echo "  $0 -n test -k sys_xxx -r github.com/user/repo -i 12345 -o sk-ant-xxx --config-only"
    exit 1
}

# Default values
COMPOSE_FILE="docker-compose.yml"
CONFIG_ONLY=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            INSTANCE_NAME="$2"
            shift 2
            ;;
        -k|--api-key)
            CCAGENT_API_KEY="$2"
            shift 2
            ;;
        -r|--repo-url)
            REPO_URL="$2"
            shift 2
            ;;
        -i|--installation-id)
            GITHUB_INSTALLATION_ID="$2"
            shift 2
            ;;
        -a|--anthropic-key)
            ANTHROPIC_API_KEY="$2"
            shift 2
            ;;
        -o|--oauth-token)
            CLAUDE_CODE_OAUTH_TOKEN="$2"
            shift 2
            ;;
        -f|--file)
            COMPOSE_FILE="$2"
            shift 2
            ;;
        -c|--config-only)
            CONFIG_ONLY=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Validate required parameters
if [ -z "$INSTANCE_NAME" ]; then
    echo "Error: Instance name is required (-n/--name)"
    usage
fi

if [ -z "$CCAGENT_API_KEY" ]; then
    echo "Error: CCAGENT_API_KEY is required (-k/--api-key)"
    usage
fi

if [ -z "$REPO_URL" ]; then
    echo "Error: Repository URL is required (-r/--repo-url)"
    usage
fi

if [ -z "$GITHUB_INSTALLATION_ID" ]; then
    echo "Error: GitHub Installation ID is required (-i/--installation-id)"
    usage
fi

# Validate authentication - must have exactly one
if [ -n "$ANTHROPIC_API_KEY" ] && [ -n "$CLAUDE_CODE_OAUTH_TOKEN" ]; then
    echo "Error: Cannot specify both Anthropic API Key and Claude Code OAuth Token"
    usage
fi

if [ -z "$ANTHROPIC_API_KEY" ] && [ -z "$CLAUDE_CODE_OAUTH_TOKEN" ]; then
    echo "Error: Must specify either Anthropic API Key (-a) or Claude Code OAuth Token (-o)"
    usage
fi

# Check if docker-compose file exists
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: Docker compose file '$COMPOSE_FILE' not found"
    exit 1
fi

# Service names
TOKEN_ROTATOR_SERVICE="ccagent-token-rotator-${INSTANCE_NAME}"
CCAGENT_SERVICE="ccagent-${INSTANCE_NAME}"
SHARED_DIR="./volumes/ccagent-${INSTANCE_NAME}"

# Function to remove existing services from docker-compose.yml
remove_existing_services() {
    echo "Checking for existing services..."
    
    # Check if services exist before trying to remove them
    if ! grep -q "^  ${TOKEN_ROTATOR_SERVICE}:" "$COMPOSE_FILE" && ! grep -q "^  ${CCAGENT_SERVICE}:" "$COMPOSE_FILE"; then
        echo "No existing services found to remove"
        return
    fi
    
    # Use sed to remove the services - much more reliable than bash parsing
    local temp_file=$(mktemp)
    
    # Copy original file to temp
    cp "$COMPOSE_FILE" "$temp_file"
    
    # Remove token-rotator service if it exists
    if grep -q "^  ${TOKEN_ROTATOR_SERVICE}:" "$temp_file"; then
        echo "Removing existing service: $TOKEN_ROTATOR_SERVICE"
        # Remove from the service line to the next service line or end of file
        sed -i "/^  ${TOKEN_ROTATOR_SERVICE}:/,/^  [a-zA-Z0-9_-]*:/{/^  ${TOKEN_ROTATOR_SERVICE}:/d; /^  [a-zA-Z0-9_-]*:/!d; /^  ${TOKEN_ROTATOR_SERVICE}:/!b; }" "$temp_file" 2>/dev/null || {
            # Fallback: use awk for more reliable removal
            awk -v service="  ${TOKEN_ROTATOR_SERVICE}:" '
                BEGIN { skip = 0 }
                $0 ~ "^  [a-zA-Z0-9_-]+:[ ]*$" {
                    if ($0 == service) {
                        skip = 1
                        next
                    } else {
                        skip = 0
                    }
                }
                !skip { print }
            ' "$temp_file" > "$temp_file.tmp" && mv "$temp_file.tmp" "$temp_file"
        }
    fi
    
    # Remove ccagent service if it exists
    if grep -q "^  ${CCAGENT_SERVICE}:" "$temp_file"; then
        echo "Removing existing service: $CCAGENT_SERVICE"
        # Use awk for reliable removal
        awk -v service="  ${CCAGENT_SERVICE}:" '
            BEGIN { skip = 0 }
            $0 ~ "^  [a-zA-Z0-9_-]+:[ ]*$" {
                if ($0 == service) {
                    skip = 1
                    next
                } else {
                    skip = 0
                }
            }
            !skip { print }
        ' "$temp_file" > "$temp_file.tmp" && mv "$temp_file.tmp" "$temp_file"
    fi
    
    # Replace the original file with the updated version
    mv "$temp_file" "$COMPOSE_FILE"
}

# Function to append services to docker-compose.yml
append_services() {
    echo "Adding services: $TOKEN_ROTATOR_SERVICE and $CCAGENT_SERVICE"
    
    # Create shared directory if it doesn't exist
    if [ ! -d "$SHARED_DIR" ]; then
        echo "Creating directory: $SHARED_DIR"
        mkdir -p "$SHARED_DIR"
        # Set ownership to 1000:1000 (requires sudo)
        echo "Setting ownership for $SHARED_DIR to 1000:1000"
        sudo chown 1000:1000 "$SHARED_DIR"
    fi
    
    # Add token-rotator service
    cat >> "$COMPOSE_FILE" << EOF

  ${TOKEN_ROTATOR_SERVICE}:
    image: preslavmihaylov/ccagent-token-rotator:latest
    container_name: ${TOKEN_ROTATOR_SERVICE}
    volumes:
      - ./cc.pem:/secrets/cc.pem:ro
      - ${SHARED_DIR}:/shared:Z
    environment:
      - GITHUB_APP_ID=1798229
      - GITHUB_INSTALLATION_ID=${GITHUB_INSTALLATION_ID}
    healthcheck:
      test: ["CMD", "test", "-f", "/shared/.config/ccagent/.env"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    restart: unless-stopped

  ${CCAGENT_SERVICE}:
    image: preslavmihaylov/ccagent-docker:latest
    container_name: ${CCAGENT_SERVICE}
    depends_on:
      ${TOKEN_ROTATOR_SERVICE}:
        condition: service_healthy
    volumes:
      - ${SHARED_DIR}:/home/ccagent
    environment:
      - REPO_URL=${REPO_URL}
      - CCAGENT_API_KEY=${CCAGENT_API_KEY}
EOF

    # Add the appropriate authentication method
    if [ -n "$ANTHROPIC_API_KEY" ]; then
        echo "      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}" >> "$COMPOSE_FILE"
    else
        echo "      - CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}" >> "$COMPOSE_FILE"
    fi

    # Add the rest of the ccagent service definition
    cat >> "$COMPOSE_FILE" << EOF
    working_dir: /workspace
    stdin_open: true
    tty: true
    restart: unless-stopped
EOF
}


# Remove existing services (if any) and add new ones
remove_existing_services
append_services

echo ""
echo "Successfully updated services for instance '$INSTANCE_NAME' in $COMPOSE_FILE"

if [ "$CONFIG_ONLY" = true ]; then
    echo ""
    echo "Config-only mode: Docker compose configuration updated, services not redeployed"
    echo "To deploy the services, run:"
    echo "  docker compose -f \"$COMPOSE_FILE\" up -d --pull always --remove-orphans"
else
    echo ""
    echo "Starting services with docker compose..."
    docker compose -f "$COMPOSE_FILE" up -d --pull always --remove-orphans

    if [ $? -eq 0 ]; then
        echo ""
        echo "Services started successfully!"
        echo "Token rotator container: '${TOKEN_ROTATOR_SERVICE}' is now running"
        echo "CCAgent container: '${CCAGENT_SERVICE}' is now running"
    else
        echo ""
        echo "Error: Failed to start services"
        exit 1
    fi
fi

