FROM alpine:latest

# Install system dependencies
RUN apk update && apk add --no-cache \
    git \
    curl \
    bash \
    ca-certificates \
    nodejs \
    npm

# Install GitHub CLI
RUN apk add --no-cache github-cli

# Install Claude binary via npm
RUN npm install -g @anthropic-ai/claude-code

# Install ccagent binary
RUN curl -fsSL -o /usr/local/bin/ccagent https://github.com/presmihaylov/ccagent/releases/download/v0.0.1/ccagent-v0.0.1-linux-x86_64 && \
    chmod +x /usr/local/bin/ccagent

# Add Claude binary to PATH
ENV PATH="/root/.local/bin:${PATH}"

# Set working directory
WORKDIR /workspace

# Default command
CMD ["/bin/bash"]