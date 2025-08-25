#!/bin/bash
set -e

# Wait for token to be available from token rotator container
echo "Waiting for GitHub token from token rotator..."
timeout=60
while [[ ! -f /home/ccagent/.config/ccagent/.env && $timeout -gt 0 ]]; do
  sleep 1
  timeout=$((timeout - 1))
done

if [[ ! -f /home/ccagent/.config/ccagent/.env ]]; then
  echo "Error: Token file not found after 60 seconds. Token rotator may not be running."
  exit 1
fi

# Source from ccagent config directory
echo "Token file found, sourcing environment..."
set -a  # automatically export all variables
source /home/ccagent/.config/ccagent/.env
set +a  # disable automatic export

# Configure git with claudecontrol identity
echo "Configuring git user..."
git config --global user.name "claudecontrol"
git config --global user.email "claudecontrol@users.noreply.github.com"

# Clone repository if REPO_URL is provided
if [[ -n "${REPO_URL:-}" ]]; then
  echo "Cloning repository: $REPO_URL"
  
  
  if [[ -z "${GH_TOKEN:-}" ]]; then
    echo "Warning: GH_TOKEN not available, trying clone without authentication"
    git clone "https://${REPO_URL}.git" repo || echo "Failed to clone repository"
  else
    # Build the authenticated URL from github.com/owner/repo format
    REPO_WITH_TOKEN="https://x-access-token:${GH_TOKEN}@${REPO_URL}.git"
    git clone "$REPO_WITH_TOKEN" repo || echo "Failed to clone repository"
  fi
  
  if [[ -d "repo" ]]; then
    cd repo
    echo "Repository cloned successfully. Current directory: $(pwd)"
    
    # Run ccagent with claude bypass permissions in foreground
    echo "Starting ccagent with claude bypass permissions..."
    exec ccagent --claude-bypass-permissions
  fi
fi

# Start interactive bash
exec /bin/bash "$@"
