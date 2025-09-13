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
    coreutils \
    make \
    go \
    gcc \
    musl-dev \
    libc-dev \
    upx

# Install GitHub CLI
RUN apk add --no-cache github-cli

# Install bun
RUN curl -fsSL https://bun.com/install | bash

# Install JS/TS tools globally
RUN npm install -g \
    typescript \
    pnpm \
    eslint \
    prettier

# Install Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install golang.org/x/tools/cmd/goimports@latest

# Add Go binaries to PATH (since go install puts them in /root/go/bin by default)
ENV PATH="/root/go/bin:${PATH}"

# Install Claude binary via npm
RUN npm install -g @anthropic-ai/claude-code

# Set ccagent version
ARG CCAGENT_VERSION=v0.0.5

# Install ccagent binary
RUN curl -fsSL -o /usr/local/bin/ccagent https://github.com/presmihaylov/ccagent/releases/download/${CCAGENT_VERSION}/ccagent-${CCAGENT_VERSION}-linux-x86_64 && \
    chmod +x /usr/local/bin/ccagent

# Add Claude binary to PATH
ENV PATH="/home/ccagent/.local/bin:${PATH}"

# Create non-root user
RUN addgroup -g 1000 ccagent && \
    adduser -D -s /bin/bash -u 1000 -G ccagent ccagent

# Copy startup script
COPY startup.sh /usr/local/bin/startup.sh
RUN chmod +x /usr/local/bin/startup.sh

# Create workspace directory and set ownership
RUN mkdir -p /workspace && chown -R ccagent:ccagent /workspace

# Switch to non-root user
USER ccagent

# Set working directory
WORKDIR /workspace

# Default command
CMD ["/usr/local/bin/startup.sh"]
