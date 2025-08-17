#!/bin/bash
set -e

# Start token rotation in background
/usr/local/bin/rotate-gh-token.sh &

# Source token environment
sleep 2  # Give token rotation time to complete initial run
if [[ -f /tmp/gh_token_env ]]; then
  source /tmp/gh_token_env
fi

# Configure git with claudecontrol identity
echo "Configuring git user..."
git config --global user.name "claudecontrol"
git config --global user.email "claudecontrol@users.noreply.github.com"

# Clone repository if REPO_URL is provided
if [[ -n "${REPO_URL:-}" ]]; then
  echo "Cloning repository: $REPO_URL"
  
  # Wait for GH_TOKEN to be available
  timeout=30
  while [[ -z "${GH_TOKEN:-}" && $timeout -gt 0 ]]; do
    echo "Waiting for GH_TOKEN to be available..."
    sleep 1
    if [[ -f /tmp/gh_token_env ]]; then
      source /tmp/gh_token_env
    fi
    timeout=$((timeout - 1))
  done
  
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
