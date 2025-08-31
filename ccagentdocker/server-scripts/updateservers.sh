#!/bin/bash

# Array of server IPs
SERVERS=(
    "143.110.239.108"
    # Add more server IPs here as needed
)

# SSH key path
SSH_KEY="$HOME/.ssh/cc"

# Scripts to upload (all .sh files except this one)
SCRIPTS=(
    "redeployall.sh"
    "redeployccagent.sh"
)

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Starting server update process..."
echo "==============================="

for SERVER in "${SERVERS[@]}"; do
    echo -e "\n${GREEN}Updating server: ${SERVER}${NC}"
    echo "-------------------------------"
    
    # Create remote directory if it doesn't exist
    echo "Creating remote directory..."
    ssh -i "$SSH_KEY" root@"$SERVER" "mkdir -p /root/scripts" 2>/dev/null
    
    # Upload each script
    for SCRIPT in "${SCRIPTS[@]}"; do
        if [ -f "$SCRIPT" ]; then
            echo "Uploading $SCRIPT..."
            scp -i "$SSH_KEY" "$SCRIPT" root@"$SERVER":/root/scripts/
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}✓ Successfully uploaded $SCRIPT${NC}"
                
                # Make script executable on remote server
                ssh -i "$SSH_KEY" root@"$SERVER" "chmod +x /root/scripts/$SCRIPT"
            else
                echo -e "${RED}✗ Failed to upload $SCRIPT${NC}"
            fi
        else
            echo -e "${RED}✗ Script $SCRIPT not found${NC}"
        fi
    done
    
    echo -e "${GREEN}Completed update for server: ${SERVER}${NC}"
done

echo -e "\n==============================="
echo -e "${GREEN}All servers updated!${NC}"
