# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o gemini-wrapper .

# Runtime stage
FROM node:20-alpine

# Install build dependencies for native modules and runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    bash \
    curl \
    wget \
    python3 \
    make \
    g++ \
    && ln -sf python3 /usr/bin/python

# Install Gemini CLI (has native dependencies that need compilation)
RUN npm install -g @google/gemini-cli && \
    gemini --version && \
    echo "✓ Gemini CLI installed successfully"

# Clean up build dependencies to reduce image size
RUN apk del python3 make g++

# Set up working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/gemini-wrapper .

# Create .gemini directory and set permissions
# Use the existing 'node' user from node:20-alpine (UID 1000, GID 1000)
RUN mkdir -p /app/.gemini && \
    chown -R node:node /app

# Switch to non-root user
USER node

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV HOME=/app
ENV GEMINI_CONFIG_DIR=/app/.gemini

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Run the application
CMD ["./gemini-wrapper"]
