#!/usr/bin/env bash
set -euo pipefail

# --- Configuration ---
APP_ID="${GITHUB_APP_ID:-}"                  
INSTALLATION_ID="${GITHUB_INSTALLATION_ID:-}"
PRIVATE_KEY_PATH="${GITHUB_APP_PRIVATE_KEY_PATH:-/workspace/cc.pem}"

if [[ -z "$APP_ID" || -z "$INSTALLATION_ID" ]]; then
  echo "Error: GITHUB_APP_ID and GITHUB_INSTALLATION_ID must be set"
  exit 1
fi

if [[ ! -f "$PRIVATE_KEY_PATH" ]]; then
  echo "Error: Private key file not found at $PRIVATE_KEY_PATH"
  exit 1
fi

rotate_token() {
  echo "$(date): Starting GitHub token rotation..."
  
  # --- Step 1: Build JWT ---
  NOW=$(date +%s)
  IAT=$((NOW - 60))            # issued at (backdated 60s for clock skew)
  EXP=$((NOW + 540))           # expires at (9 minutes ahead)

  HEADER=$(jq -nc '{"alg":"RS256","typ":"JWT"}')
  PAYLOAD=$(jq -nc --arg iat "$IAT" --arg exp "$EXP" --arg iss "$APP_ID" \
    '{iat:($iat|tonumber), exp:($exp|tonumber), iss:($iss|tonumber)}')

  b64url() { openssl base64 -e -A | tr '+/' '-_' | tr -d '='; }

  HEADER_B64=$(echo -n "$HEADER" | b64url)
  PAYLOAD_B64=$(echo -n "$PAYLOAD" | b64url)
  UNSIGNED="${HEADER_B64}.${PAYLOAD_B64}"

  SIGNATURE=$(echo -n "$UNSIGNED" | \
    openssl dgst -sha256 -sign "$PRIVATE_KEY_PATH" | b64url)

  JWT="${UNSIGNED}.${SIGNATURE}"

  # --- Step 2: Exchange JWT for installation token ---
  RESPONSE=$(curl -s -X POST \
    -H "Authorization: Bearer $JWT" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/app/installations/${INSTALLATION_ID}/access_tokens")

  TOKEN=$(echo "$RESPONSE" | jq -r .token)
  EXPIRES=$(echo "$RESPONSE" | jq -r .expires_at)

  if [[ "$TOKEN" == "null" ]]; then
    echo "Error: Failed to fetch token"
    echo "$RESPONSE"
    return 1
  fi

  # Set the token as environment variable
  export GH_TOKEN="$TOKEN"
  echo "export GH_TOKEN=\"$TOKEN\"" > /tmp/gh_token_env
  
  echo "$(date): Token rotated successfully. Expires at: $EXPIRES"
  return 0
}

# Source the token environment if it exists
if [[ -f /tmp/gh_token_env ]]; then
  source /tmp/gh_token_env
fi

# If this script is called with "rotate" argument, just rotate once and exit
if [[ "${1:-}" == "rotate" ]]; then
  rotate_token
  exit $?
fi

# Initial token rotation
rotate_token

# Background rotation every 45 minutes
while true; do
  sleep 2700  # 45 minutes = 2700 seconds
  rotate_token || echo "$(date): Token rotation failed, will retry in 45 minutes"
done
