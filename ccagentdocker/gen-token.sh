#!/usr/bin/env bash
set -euo pipefail

# --- Hardcoded config (edit these) ---
APP_ID="1798229"
INSTALLATION_ID="82566909"
PRIVATE_KEY_PATH="./cc.pem"

# --- Sanity checks ---
[[ -f "$PRIVATE_KEY_PATH" ]] || { echo "Error: Private key not found at $PRIVATE_KEY_PATH"; exit 1; }

# --- Build JWT ---
NOW=$(date +%s)
IAT=$((NOW - 60))      # backdate 60s for clock skew
EXP=$((NOW + 540))     # 9 minutes ahead

HEADER=$(jq -nc '{"alg":"RS256","typ":"JWT"}')
PAYLOAD=$(jq -nc --arg iat "$IAT" --arg exp "$EXP" --arg iss "$APP_ID" \
  '{iat:($iat|tonumber), exp:($exp|tonumber), iss:($iss|tonumber)}')

b64url() { openssl base64 -e -A | tr '+/' '-_' | tr -d '='; }

HEADER_B64=$(printf %s "$HEADER" | b64url)
PAYLOAD_B64=$(printf %s "$PAYLOAD" | b64url)
UNSIGNED="${HEADER_B64}.${PAYLOAD_B64}"

SIGNATURE=$(printf %s "$UNSIGNED" | openssl dgst -sha256 -sign "$PRIVATE_KEY_PATH" | b64url)
JWT="${UNSIGNED}.${SIGNATURE}"

# --- Exchange JWT for installation token ---
RESPONSE=$(curl -sS -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/app/installations/${INSTALLATION_ID}/access_tokens")

TOKEN=$(printf %s "$RESPONSE" | jq -r .token)

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "Error: Failed to fetch token"
  echo "$RESPONSE"
  exit 1
fi

# --- Print token only ---
echo "$TOKEN"

