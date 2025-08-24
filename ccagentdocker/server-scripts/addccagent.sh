#!/bin/bash

# Script to add new ccagent services to docker-compose.yml

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Required options:"
    echo "  -n, --name NAME                    Service name prefix (e.g., 'ccagent-test')"
    echo "  -k, --api-key KEY                  CCAGENT_API_KEY"
    echo "  -r, --repo-url URL                 Repository URL (e.g., 'github.com/user/repo')"
    echo "  -i, --installation-id ID           GitHub Installation ID"
    echo ""
    echo "Authentication (choose one):"
    echo "  -a, --anthropic-key KEY            Anthropic API Key"
    echo "  -o, --oauth-token TOKEN            Claude Code OAuth Token"
    echo ""
    echo "Optional:"
    echo "  -c, --count COUNT                  Number of instances to create (default: 1)"
    echo "  -f, --file FILE                    Docker compose file (default: docker-compose.yml)"
    echo ""
    echo "Example:"
    echo "  $0 -n ccagent-prod -k sys_xxx -r github.com/user/repo -i 12345 -o sk-ant-xxx -c 2"
    exit 1
}

# Default values
COUNT=1
COMPOSE_FILE="docker-compose.yml"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            SERVICE_NAME="$2"
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
        -c|--count)
            COUNT="$2"
            shift 2
            ;;
        -f|--file)
            COMPOSE_FILE="$2"
            shift 2
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
if [ -z "$SERVICE_NAME" ]; then
    echo "Error: Service name is required (-n/--name)"
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

# Validate COUNT is a positive integer
if ! [[ "$COUNT" =~ ^[1-9][0-9]*$ ]]; then
    echo "Error: Count must be a positive integer"
    exit 1
fi

# Function to append service to docker-compose.yml
append_service() {
    local suffix="$1"
    local service_name="${SERVICE_NAME}${suffix}"
    
    echo "Adding service: $service_name"
    
    # Create directory for the service if it doesn't exist
    if [ ! -d "./${service_name}" ]; then
        echo "Creating directory: ./${service_name}"
        mkdir -p "./${service_name}"
    fi
    
    # Set ownership to 1000:1000 (requires sudo)
    echo "Setting ownership for ./${service_name} to 1000:1000"
    sudo chown 1000:1000 "./${service_name}"
    
    # Start building the service definition
    cat >> "$COMPOSE_FILE" << EOF

  ${service_name}:
    image: preslavmihaylov/ccagent-docker:latest
    container_name: ${service_name}
    restart: always
    environment:
      # GitHub App Configuration
      - GITHUB_APP_ID=1798229
      - GITHUB_INSTALLATION_ID=${GITHUB_INSTALLATION_ID}
      - GITHUB_APP_PRIVATE_KEY_PATH=/workspace/cc.pem
      - REPO_URL=${REPO_URL}
      - CCAGENT_API_KEY=${CCAGENT_API_KEY}
EOF

    # Add the appropriate authentication method
    if [ -n "$ANTHROPIC_API_KEY" ]; then
        echo "      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}" >> "$COMPOSE_FILE"
    else
        echo "      - CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}" >> "$COMPOSE_FILE"
    fi

    # Add the rest of the service definition
    cat >> "$COMPOSE_FILE" << EOF
    volumes:
      # Mount your GitHub App private key
      - ./cc.pem:/workspace/cc.pem:ro
      - ./${service_name}:/home/ccagent
    working_dir: /workspace
    stdin_open: true
    tty: true
EOF
}

# Backup the original docker-compose file
cp "$COMPOSE_FILE" "${COMPOSE_FILE}.bak.$(date +%Y%m%d_%H%M%S)"
echo "Created backup: ${COMPOSE_FILE}.bak.$(date +%Y%m%d_%H%M%S)"

# Add services to docker-compose.yml
if [ "$COUNT" -eq 1 ]; then
    # Single instance - no suffix
    append_service ""
else
    # Multiple instances - add numeric suffix
    for i in $(seq 1 $COUNT); do
        append_service "-$i"
    done
fi

echo ""
echo "Successfully added $COUNT service(s) to $COMPOSE_FILE"
echo ""
echo "Starting services with docker compose..."
docker compose -f "$COMPOSE_FILE" up -d --pull always --remove-orphans

if [ $? -eq 0 ]; then
    echo ""
    echo "Services started successfully!"
    if [ "$COUNT" -eq 1 ]; then
        echo "Container '${SERVICE_NAME}' is now running"
    else
        echo "Containers '${SERVICE_NAME}-1' through '${SERVICE_NAME}-${COUNT}' are now running"
    fi
else
    echo ""
    echo "Error: Failed to start services"
    exit 1
fi