# Gemini Wrapper API

[![Docker Hub](https://img.shields.io/docker/v/antiantiops/gemini-wrapper?label=Docker%20Hub&logo=docker)](https://hub.docker.com/r/antiantiops/gemini-wrapper)
[![Docker Pulls](https://img.shields.io/docker/pulls/antiantiops/gemini-wrapper)](https://hub.docker.com/r/antiantiops/gemini-wrapper)
[![Docker Image Size](https://img.shields.io/docker/image-size/antiantiops/gemini-wrapper/latest)](https://hub.docker.com/r/antiantiops/gemini-wrapper)

A Go REST API wrapper for Google's Gemini CLI using Echo framework. This service provides a simple HTTP interface to interact with Gemini AI programmatically.

🐳 **Pre-built Docker images available on Docker Hub**: https://hub.docker.com/r/antiantiops/gemini-wrapper

## Features

- ✅ REST API interface for Gemini CLI
- ✅ **Pre-built multi-platform Docker images** (amd64, arm64)
- ✅ Host-based authentication (no API keys in config)
- ✅ Built with Go and Echo framework
- ✅ Automatic TUI interaction handling
- ✅ Thread-safe request processing
- ✅ Health check endpoint
- ✅ CORS enabled

## Quick Start (Using Docker Hub)

### Step 1: Authenticate Gemini CLI on Your Host

```bash
# Install Gemini CLI (requires Node.js 20+)
npm install -g @google/gemini-cli

# Authenticate (opens browser for Google login)
gemini
```

This stores your credentials in `~/.gemini` folder.

### Step 2: Pull and Run from Docker Hub

**Linux/Mac:**
```bash
docker pull antiantiops/gemini-wrapper:latest

docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  --name gemini-wrapper \
  antiantiops/gemini-wrapper:latest
```

**Windows (PowerShell):**
```powershell
docker pull antiantiops/gemini-wrapper:latest

docker run -d -p 8080:8080 `
  -v ${env:USERPROFILE}\.gemini:/app/.gemini `
  --name gemini-wrapper `
  antiantiops/gemini-wrapper:latest
```

**Note**: The volume is mounted as **read-write** so Gemini CLI can automatically refresh authentication tokens.

### Step 3: Test the API

```bash
# Health check
curl http://localhost:8080/

# Ask a question
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "What is 2+2?"}'
```

**Response:**
```json
{
  "answer": "2+2 equals 4."
}
```

That's it! 🎉

## Using Docker Compose

### Create docker-compose.yml

**Linux/Mac:**
```yaml
version: '3.8'

services:
  gemini-wrapper:
    image: antiantiops/gemini-wrapper:latest
    container_name: gemini-wrapper
    ports:
      - "8080:8080"
    volumes:
      - ${HOME}/.gemini:/app/.gemini  # Read-write for token renewal
    restart: unless-stopped
```

**Windows:**
```yaml
version: '3.8'

services:
  gemini-wrapper:
    image: antiantiops/gemini-wrapper:latest
    container_name: gemini-wrapper
    ports:
      - "8080:8080"
    volumes:
      - ${USERPROFILE}/.gemini:/app/.gemini  # Read-write for token renewal
    restart: unless-stopped
```

### Run with Docker Compose

```bash
# Start
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

## Docker Hub Images

🐳 **Repository**: https://hub.docker.com/r/antiantiops/gemini-wrapper

### Available Tags

| Tag | Description | Recommended For |
|-----|-------------|-----------------|
| `latest` | Latest stable build from main branch | Development, testing |
| `1.0.0` | Specific version (when released) | Production (pin versions) |
| `1.0` | Major.minor version | Auto-patch updates |
| `1` | Major version | Auto-minor updates |

### Multi-Platform Support

Images are automatically built for multiple architectures:
- **linux/amd64** - Intel/AMD processors (x86_64)
- **linux/arm64** - ARM processors (Apple M1/M2, Raspberry Pi, AWS Graviton)

Docker automatically selects the correct architecture for your platform.

### Image Size

Approximately **350-400 MB** (includes Node.js 20, Gemini CLI, and Go application)

## API Endpoints

### Health Check

```http
GET /
```

**Response:**
```json
{
  "message": "Gemini Wrapper API",
  "status": "running"
}
```

### Ask Question

```http
POST /api/ask
Content-Type: application/json
```

**Request Body:**
```json
{
  "question": "Your question here",
  "model": "gemini-3-flash"
}
```

**Parameters:**
- `question` (required): Your question or prompt
- `model` (optional): Gemini model to use

**Response (Success):**
```json
{
  "answer": "The AI's response here"
}
```

**Response (Error):**
```json
{
  "error": "Error message"
}
```

## Usage Examples

### cURL

```bash
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Explain quantum computing in simple terms"}'
```

### Python

```python
import requests

response = requests.post(
    "http://localhost:8080/api/ask",
    json={"question": "What is machine learning?"}
)

result = response.json()
print(result["answer"])
```

### JavaScript/Node.js

```javascript
const response = await fetch('http://localhost:8080/api/ask', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    question: 'Explain REST APIs'
  })
});

const data = await response.json();
console.log(data.answer);
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

func main() {
    reqBody := map[string]string{
        "question": "What is Go programming language?",
    }
    
    jsonData, _ := json.Marshal(reqBody)
    resp, _ := http.Post(
        "http://localhost:8080/api/ask",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    
    var result map[string]string
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Println(result["answer"])
}
```

## Authentication

This application uses **host-based authentication**:

1. You authenticate Gemini CLI once on your host machine
2. The Docker container mounts your `~/.gemini` credentials folder
3. Container uses your authenticated session automatically

**Benefits:**
- ✅ No API keys in environment variables
- ✅ Persistent authentication with automatic token renewal
- ✅ Secure credential management
- ✅ Easy to use

### Important: Volume Mount Permissions

The `.gemini` folder is mounted as **read-write** (not read-only) because Gemini CLI needs to automatically refresh OAuth tokens. Without write access, token renewal will fail and the container will stop working when tokens expire.

```bash
# Correct - allows token renewal
-v ~/.gemini:/app/.gemini

# Wrong - blocks token renewal
-v ~/.gemini:/app/.gemini:ro  # Don't use :ro!
```

See [VOLUME_MOUNT_EXPLANATION.md](VOLUME_MOUNT_EXPLANATION.md) for detailed explanation.

## Prerequisites

- **Docker** installed and running
- **Node.js 20+** (for Gemini CLI authentication)
- **Gemini CLI** installed: `npm install -g @google/gemini-cli`
- **Authenticated** with Gemini: Run `gemini` and complete login

## Common Commands

### Pull Latest Image

```bash
docker pull antiantiops/gemini-wrapper:latest
```

### Run Container

```bash
# Linux/Mac
docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  --name gemini-wrapper \
  antiantiops/gemini-wrapper:latest

# Windows
docker run -d -p 8080:8080 `
  -v ${env:USERPROFILE}\.gemini:/app/.gemini `
  --name gemini-wrapper `
  antiantiops/gemini-wrapper:latest
```

### View Logs

```bash
docker logs -f gemini-wrapper
```

### Stop Container

```bash
docker stop gemini-wrapper
```

### Start Container

```bash
docker start gemini-wrapper
```

### Remove Container

```bash
docker rm -f gemini-wrapper
```

### Update to Latest

```bash
docker stop gemini-wrapper
docker rm gemini-wrapper
docker pull antiantiops/gemini-wrapper:latest
docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  --name gemini-wrapper \
  antiantiops/gemini-wrapper:latest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP server port |
| `HOME` | /app | Home directory |
| `GEMINI_CONFIG_DIR` | /app/.gemini | Gemini config directory |

Override when running:
```bash
docker run -d -p 9000:9000 \
  -e PORT=9000 \
  -v ~/.gemini:/app/.gemini \
  antiantiops/gemini-wrapper:latest
```

## Troubleshooting

### "Authentication failed" or "No credentials found"

**Solution:**
```bash
# Verify credentials exist
ls ~/.gemini  # Linux/Mac
dir %USERPROFILE%\.gemini  # Windows

# Re-authenticate if needed
gemini
```

### "Port already in use"

**Solution:**
```bash
# Use different port
docker run -d -p 9000:8080 ...
```

### "Permission denied" on credentials

**Solution (Linux/Mac):**
```bash
chmod -R 755 ~/.gemini
```

### Container won't start

**Solution:**
```bash
# Check logs
docker logs gemini-wrapper

# Verify Docker is running
docker ps

# Check if port is available
netstat -an | grep 8080
```

## Building from Source (Alternative)

If you prefer to build locally instead of using Docker Hub:

### Prerequisites
- Go 1.25+
- Node.js 20+
- Docker

### Build and Run

```bash
# Clone repository
git clone <your-repo-url>
cd gemini-wrapper

# Build Docker image
docker build -t gemini-wrapper .

# Run
docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  gemini-wrapper
```

## Project Structure

```
gemini-wrapper/
├── main.go                 # HTTP server and API routes
├── gemini_service.go       # Gemini CLI interaction logic
├── gemini_service_test.go  # Unit tests
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.yml      # Container orchestration
├── go.mod                  # Go dependencies
└── README.md              # This file
```

## Technology Stack

- **Language**: Go 1.25
- **Framework**: Echo v4 (HTTP)
- **PTY Library**: creack/pty (pseudo-terminal)
- **Runtime**: Node.js 20 (for Gemini CLI)
- **Container**: Docker (Alpine Linux)

## Security

- ✅ Non-root user (runs as `node` user)
- ✅ Read-only credential mount
- ✅ No API keys in environment variables
- ✅ Automatic token refresh by Gemini CLI
- ✅ CORS can be configured

## Performance

- **Throughput**: ~1 request per 5-10 seconds per instance
- **Latency**: 5-15 seconds average (depends on Gemini API)
- **Memory**: ~100-200 MB per instance
- **CPU**: Low (mostly I/O waiting)

**Scaling**: Run multiple containers behind a load balancer for higher throughput.

## Production Deployment

### Recommended Setup

```yaml
version: '3.8'

services:
  gemini-wrapper:
    image: antiantiops/gemini-wrapper:1.0.0  # Pin to specific version
    container_name: gemini-wrapper
    ports:
      - "8080:8080"
    volumes:
      - ${HOME}/.gemini:/app/.gemini  # Read-write for token renewal
    restart: always
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Best Practices

1. **Pin versions in production**: Use `antiantiops/gemini-wrapper:1.0.0` instead of `:latest`
2. **Set resource limits**: Prevent container from consuming too many resources
3. **Use health checks**: Automatic restart on failures
4. **Monitor logs**: `docker logs -f gemini-wrapper`
5. **Regular updates**: Pull new versions and test before deploying
6. **Reverse proxy**: Use nginx or Caddy for SSL/TLS termination

## Support & Contributing

- **Docker Hub**: https://hub.docker.com/r/antiantiops/gemini-wrapper
- **Issues**: Report issues on GitHub repository
- **Documentation**: See repository for detailed guides

## License

MIT License - Free to use and modify

---

## Quick Reference

```bash
# Install Gemini CLI and authenticate
npm install -g @google/gemini-cli
gemini

# Pull and run
docker pull antiantiops/gemini-wrapper:latest
docker run -d -p 8080:8080 -v ~/.gemini:/app/.gemini antiantiops/gemini-wrapper:latest

# Test
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Hello!"}'

# View logs
docker logs -f gemini-wrapper
```

---

**Ready to use! Pull the image and start querying Gemini AI through a REST API.** 🚀
