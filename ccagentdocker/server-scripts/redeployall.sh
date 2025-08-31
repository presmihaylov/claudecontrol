#!/bin/bash

# Script to redeploy all services in docker-compose.yml
# This script simply runs docker compose up with the correct flags

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Optional:"
    echo "  -f, --file FILE                    Docker compose file (default: docker-compose.yml)"
    echo "  -h, --help                         Show this help message"
    echo ""
    echo "Example:"
    echo "  $0"
    echo "  $0 -f my-docker-compose.yml"
    exit 1
}

# Default values
COMPOSE_FILE="docker-compose.yml"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
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

# Check if docker-compose file exists
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: Docker compose file '$COMPOSE_FILE' not found"
    exit 1
fi

echo "Starting all services with docker compose from $COMPOSE_FILE..."
docker compose -f "$COMPOSE_FILE" up -d --pull always --remove-orphans

if [ $? -eq 0 ]; then
    echo ""
    echo "All services started successfully!"
    echo ""
    echo "Running services:"
    docker compose -f "$COMPOSE_FILE" ps
else
    echo ""
    echo "Error: Failed to start services"
    exit 1
fi