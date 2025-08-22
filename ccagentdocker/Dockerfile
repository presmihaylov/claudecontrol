FROM alpine:latest

# Install system dependencies
RUN apk update && apk add --no-cache \
    git \
    curl \
    bash \
    ca-certificates \
    nodejs \
    npm \
    jq \
    openssl \
    coreutils

# Install GitHub CLI
RUN apk add --no-cache github-cli

# Install Claude binary via npm
RUN npm install -g @anthropic-ai/claude-code

# Install ccagent binary
RUN curl -fsSL -o /usr/local/bin/ccagent https://github.com/presmihaylov/ccagent/releases/download/v0.0.3/ccagent-v0.0.3-linux-x86_64 && \
    chmod +x /usr/local/bin/ccagent

# Add Claude binary to PATH
ENV PATH="/home/ccagent/.local/bin:${PATH}"

# Create non-root user
RUN addgroup -g 1000 ccagent && \
    adduser -D -s /bin/bash -u 1000 -G ccagent ccagent

# Copy scripts
COPY rotate-gh-token.sh /usr/local/bin/rotate-gh-token.sh
COPY startup.sh /usr/local/bin/startup.sh
RUN chmod +x /usr/local/bin/rotate-gh-token.sh /usr/local/bin/startup.sh

# Create workspace directory and set ownership
RUN mkdir -p /workspace && chown -R ccagent:ccagent /workspace

# Switch to non-root user
USER ccagent

# Set working directory
WORKDIR /workspace

# Default command
CMD ["/usr/local/bin/startup.sh"]
