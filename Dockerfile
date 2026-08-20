# Build stage
FROM golang:1.27-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -o gemini-wrapper .

# Runtime stage
FROM node:24-bookworm-slim

# Install build dependencies for native modules and runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    bash \
    curl \
    wget \
    git \
    python3 \
    make \
    g++ \
    && ln -sf python3 /usr/bin/python \
    && rm -rf /var/lib/apt/lists/*

# Install Antigravity CLI
# Official Linux install: https://antigravity.google/cli/install.sh
# Install into /usr/local/bin so `agy` is on PATH for both the build check and
# at runtime (where HOME=/app, so the script's default ~/.local/bin is unusable).
#
# IMPORTANT: The install script skips download when the binary already exists.
# We rm -f first so every build gets the latest version. The ARG below also
# busts the Docker layer cache when the Release Watch workflow bumps the pin.
ARG AGY_VERSION=latest
RUN rm -f /usr/local/bin/agy && \
  curl -fsSL https://antigravity.google/cli/install.sh | bash -s -- --dir /usr/local/bin && \
  /usr/local/bin/agy --version && \
  echo "✓ Antigravity CLI installed successfully (expected: ${AGY_VERSION})"

# Set up working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/gemini-wrapper .

# Create Antigravity config directory
# Note: agy reads its config/credentials from $HOME/.gemini (HOME=/app here).
# Running as root to avoid permission issues with mounted volumes
RUN mkdir -p /app/.gemini

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV HOME=/app
ENV ANTIGRAVITY_CONFIG_DIR=/app/.gemini
ENV ANTIGRAVITY_CLI_COMMAND=agy

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider --method=GET http://localhost:8080/ || exit 1

# Run as root user to avoid permission issues with mounted volumes
# Note: Running as root is simpler but less secure than using a non-root user
CMD ["./gemini-wrapper"]
