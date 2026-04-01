# Gemini Wrapper API

[![Docker Hub](https://img.shields.io/docker/v/antiantiops/gemini-wrapper?label=Docker%20Hub&logo=docker)](https://hub.docker.com/r/antiantiops/gemini-wrapper)
[![Docker Pulls](https://img.shields.io/docker/pulls/antiantiops/gemini-wrapper)](https://hub.docker.com/r/antiantiops/gemini-wrapper)
[![Docker Image Size](https://img.shields.io/docker/image-size/antiantiops/gemini-wrapper/latest)](https://hub.docker.com/r/antiantiops/gemini-wrapper)

A Go REST API wrapper for Google's Gemini CLI. Provides a simple HTTP interface to interact with Gemini AI.

🐳 **Pre-built Docker images**: https://hub.docker.com/r/antiantiops/gemini-wrapper

---

## 🚨 READ THIS FIRST

### You Do NOT Need to Install Gemini CLI on Your Computer!

**❌ WRONG (Traditional Method):**
```bash
# DON'T DO THIS - You don't need to install on localhost!
npm install -g @google/gemini-cli
gemini
```

**✅ CORRECT (Our Method):**
```bash
# Just start the container - Gemini CLI is already inside!
docker run -d -p 8080:8080 -v ~/.gemini:/app/.gemini --name gemini-wrapper antiantiops/gemini-wrapper:latest

# Then authenticate INSIDE the container
docker exec -it gemini-wrapper sh -c 'gemini'
# Select "1. Login with Google" (NOT API Key!)
```

**Why our method is better:**
- ✅ No Node.js installation on your computer
- ✅ No npm packages on your computer  
- ✅ No Gemini CLI installation on your computer
- ✅ Everything isolated in Docker
- ✅ Only Docker required

---

## 🎯 How It Works

**You do NOT need to install anything on your computer except Docker!**

```
┌─────────────────────────────────────────────────────────────┐
│                    YOUR COMPUTER (Host)                      │
│                                                              │
│  1. Create empty folder: ~/.gemini                          │
│  2. Run Docker container with mount                         │
│                                                              │
└──────────────────────────────────────────────────────────────┘
                           ↓ Mount
┌─────────────────────────────────────────────────────────────┐
│                   DOCKER CONTAINER                           │
│                                                              │
│  • Gemini CLI pre-installed ✅                               │
│  • Node.js pre-installed ✅                                  │
│  • Go application pre-installed ✅                           │
│                                                              │
│  3. You run: gemini (inside container)                      │
│  4. Select "1. Login with Google"                           │
│  5. Authenticate via browser OAuth                          │
│  6. Credentials saved to /app/.gemini                       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
                           ↓ Mount (bidirectional)
┌─────────────────────────────────────────────────────────────┐
│                    YOUR COMPUTER (Host)                      │
│                                                              │
│  7. Credentials appear in: ~/.gemini ✅                      │
│  8. Container can now access Google Gemini API ✅            │
│  9. Your REST API is ready! ✅                               │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Key Points:**
- ✅ **No localhost installation** - Everything runs in Docker
- ✅ **Authenticate in container** - Not on your computer
- ✅ **Must use "Login with Google"** - API Key option won't work
- ✅ **Credentials shared via mount** - Saved to both container and host

---

## ⚡ Quick Start (3 Steps)

### Prerequisites

**✅ Only Docker is required!**

**❌ You do NOT need to:**
- Install Node.js on your computer
- Install Gemini CLI on your computer
- Install npm on your computer

**Everything is already inside the Docker container!**

---

### Step 1: Create Empty Folder and Start Container

**What happens:** Create an empty folder for credentials, then start the container with this folder mounted.

**Linux/Mac:**
```bash
# Create empty folder for credentials
mkdir -p ~/.gemini

# Start container with mount
docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  --name gemini-wrapper \
  antiantiops/gemini-wrapper:latest
```

**Windows (PowerShell):**
```powershell
# Create empty folder for credentials
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.gemini"

# Start container with mount
docker run -d -p 8080:8080 `
  -v ${env:USERPROFILE}\.gemini:/app/.gemini `
  --name gemini-wrapper `
  antiantiops/gemini-wrapper:latest
```

**What this does:**
- Creates an empty `~/.gemini` folder on your computer
- Starts the container
- Mounts `~/.gemini` (host) to `/app/.gemini` (container)
- When you authenticate in the container, credentials are saved to both places

---

### Step 2: Authenticate INSIDE the Container

**Important:** You authenticate **INSIDE the running Docker container**, not on your computer.

Run this command to enter the container and start authentication:

```bash
docker exec -it gemini-wrapper sh -c 'export HOME=/app && export GEMINI_CONFIG_DIR=/app/.gemini && cd /app && gemini'
```

**You'll see this menu:**

```
How would you like to authenticate for this project?

  ● 1. Login with Google         ← Type "1" and press Enter
    2. Use Gemini API Key         ← DO NOT select this
    3. Vertex AI                  ← DO NOT select this

No authentication method selected.
```

⚠️ **CRITICAL: You MUST type `1` and press Enter**

**Why "Login with Google" only?**
- ✅ Option 1 (Login with Google) - Works with this project
- ❌ Option 2 (Gemini API Key) - Will NOT work
- ❌ Option 3 (Vertex AI) - For enterprise Google Cloud only

**After selecting "1", follow these steps:**

1. Terminal shows a long URL: `https://accounts.google.com/o/oauth2/v2/auth?...`
2. **Copy the entire URL**
3. **Open it in your browser** (on your host computer)
4. **Sign in with Google** and grant permissions
5. Browser shows an authorization code
6. **Copy the authorization code**
7. **Go back to the container terminal** and paste the code
8. **Press Enter**
9. You'll see "Authentication successful!" ✓

**What happened:**
- You authenticated inside the container
- Credentials were saved to `/app/.gemini` (inside container)
- Because `/app/.gemini` is mounted to `~/.gemini` (on your computer)
- The credentials are now available on both your computer AND in the container

**Restart the container:**
```bash
docker restart gemini-wrapper
```

---

### Step 3: Test the API

```bash
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

**✅ That's it! Your Gemini API is ready to use.**

---

## 📝 Quick Summary

### What You Just Did:

1. ✅ Created empty folder: `~/.gemini` on your computer
2. ✅ Started Docker container with folder mounted
3. ✅ Ran `gemini` command **INSIDE the container** (not on your computer!)
4. ✅ Selected **"1. Login with Google"** (not API Key!)
5. ✅ Authenticated via browser OAuth
6. ✅ Credentials saved to container's `/app/.gemini`
7. ✅ Credentials automatically appear in your `~/.gemini` (via mount)
8. ✅ Restarted container
9. ✅ API is now working!

### What You Did NOT Do:

- ❌ Install Node.js on your computer
- ❌ Install npm on your computer
- ❌ Install Gemini CLI on your computer (`npm install -g @google/gemini-cli`)
- ❌ Run `gemini` command on your computer
- ❌ Use "Gemini API Key" option

### Why This Approach?

**Everything happens inside Docker:**
- Gemini CLI is already installed in the container
- You authenticate inside the container
- Credentials are shared between container and host via mount
- Your computer stays clean (no extra installations)

---

## 📡 API Usage

### Simple API (Recommended)

```bash
# Basic request
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "What is machine learning?"}'

# With specific model
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Explain quantum computing",
    "model": "gemini-2.5-flash"
  }'
```

**Response:**
```json
{
  "answer": "Machine learning is a subset of artificial intelligence..."
}
```

### Gemini API Compatible Format

```bash
curl -X POST http://localhost:8080/v1beta/models/gemini-2.5-flash \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "parts": [
          {"text": "What is machine learning?"}
        ]
      }
    ]
  }'
```

**Response:**
```json
{
  "model": "gemini-2.5-flash",
  "candidates": [
    {
      "content": {
        "parts": [
          {"text": "Machine learning is..."}
        ]
      }
    }
  ]
}
```

---

## OpenAI-Compatible API

This backend exposes OpenAI-compatible endpoints and forwards generation traffic to Gemini CLI:

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/completions`

Example request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

---

## 🎯 Available Models

| Model | Speed | Quality | Best For | Cost |
|-------|-------|---------|----------|------|
| `gemini-2.5-flash-lite` | ⚡⚡⚡ Fastest | ⭐⭐ Good | Quick answers, chat | Lowest |
| `gemini-2.5-flash` | ⚡⚡ Fast | ⭐⭐⭐ Great | Most tasks (default) | Low |
| `gemini-2.5-pro` | ⚡ Slower | ⭐⭐⭐⭐ Best | Complex tasks, research | Higher |

**Examples:**

```bash
# Fast and cheap (flash-lite)
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Hi!", "model": "gemini-2.5-flash-lite"}'

# Balanced (flash) - DEFAULT if you don't specify model
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Explain Docker"}'

# High quality (pro)
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Write a research paper on AI", "model": "gemini-2.5-pro"}'
```

---

## 🔧 Common Commands

### Container Management

```bash
# View logs
docker logs -f gemini-wrapper

# Restart container
docker restart gemini-wrapper

# Stop container
docker stop gemini-wrapper

# Start container
docker start gemini-wrapper

# Remove container
docker rm -f gemini-wrapper

# Update to latest image
docker pull antiantiops/gemini-wrapper:latest
docker rm -f gemini-wrapper
# Then run Step 1 again
```

### Health Check

```bash
# Check if API is running
curl http://localhost:8080/

# Response: {"message":"Gemini Wrapper API","status":"running"}
```

---

## 💡 Troubleshooting

### ❌ Common Mistake: Installing on localhost

**Symptom:** Trying to run `npm install -g @google/gemini-cli` or `gemini` on your computer

**Why this is wrong:**
- You don't need to install anything on your computer
- Gemini CLI is already inside the Docker container
- Authentication happens inside the container

**Correct approach:**
```bash
# Don't install on localhost - just run these:
docker exec -it gemini-wrapper sh -c 'export HOME=/app && gemini'
# Then authenticate inside container
```

---

### Issue: "authentication required" error

**Cause:** Not authenticated yet, or credentials expired

**Solution:** Authenticate inside container:
```bash
docker exec -it gemini-wrapper sh -c 'export HOME=/app && export GEMINI_CONFIG_DIR=/app/.gemini && cd /app && gemini'
```

**IMPORTANT:** When you see the menu, type **`1`** to select "Login with Google"

---

### Issue: Selected "2. Use Gemini API Key" by mistake

**Symptom:** Authentication seems to work but API returns errors

**Cause:** You selected wrong option - this project requires "Login with Google"

**Solution:** Remove credentials and re-authenticate with correct option:
```bash
# Remove wrong credentials
rm -rf ~/.gemini/*

# Re-authenticate INSIDE container
docker exec -it gemini-wrapper sh -c 'export HOME=/app && export GEMINI_CONFIG_DIR=/app/.gemini && cd /app && gemini'

# This time, select "1. Login with Google" (NOT "2. Use Gemini API Key")

# Restart container
docker restart gemini-wrapper
```

---

### Issue: Container starts but no response

**Solution:** Check logs:
```bash
docker logs gemini-wrapper
```

Look for authentication errors or model errors.

---

### Issue: Want to use a different Google account

**Solution:** Remove credentials and re-authenticate:
```bash
# Remove old credentials
rm -rf ~/.gemini/*

# Re-authenticate inside container
docker exec -it gemini-wrapper sh -c 'export HOME=/app && export GEMINI_CONFIG_DIR=/app/.gemini && cd /app && gemini'
# Select "1. Login with Google"
# Use different Google account in browser

# Restart
docker restart gemini-wrapper
```

---

### Issue: Model not found error

**Cause:** Using preview or unavailable models

**Solution:** Use stable models only:
- ✅ `gemini-2.5-flash-lite`
- ✅ `gemini-2.5-flash`
- ✅ `gemini-2.5-pro`

Avoid preview models like `gemini-3-pro` (they may not be available yet).

**Example:**
```bash
# Good - uses stable model
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Hi", "model": "gemini-2.5-flash"}'

# Bad - preview model may not exist
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Hi", "model": "gemini-3-pro"}'
```

---

## 🐳 Using Docker Compose (Optional)

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  gemini-wrapper:
    image: antiantiops/gemini-wrapper:latest
    container_name: gemini-wrapper
    ports:
      - "8080:8080"
    volumes:
      - ${HOME}/.gemini:/app/.gemini  # Linux/Mac
      # - ${USERPROFILE}/.gemini:/app/.gemini  # Windows (uncomment this, comment above)
    restart: unless-stopped
```

Run:
```bash
docker-compose up -d
```

---

## 🛠️ Building from Source (Advanced)

```bash
# Clone repository
git clone https://github.com/yourusername/gemini-wrapper.git
cd gemini-wrapper

# Build Docker image
docker build -t gemini-wrapper .

# Run
docker run -d -p 8080:8080 -v ~/.gemini:/app/.gemini --name gemini-wrapper gemini-wrapper
```

---

## 📚 Technical Details

### Features

- ✅ REST API interface for Gemini CLI
- ✅ Two API formats (Simple + Gemini API compatible)
- ✅ Headless mode (clean JSON responses)
- ✅ Multi-platform Docker images (amd64, arm64)
- ✅ OAuth-based authentication
- ✅ Built with Go and Echo framework
- ✅ Thread-safe request processing

### Architecture

```
Client → REST API (Echo/Go) → Gemini CLI → Google Gemini API
```

The wrapper executes `gemini --prompt "question" --output-format json` for each request and returns the structured response.

### Multi-Platform Support

Images are automatically built for:
- **linux/amd64** - Intel/AMD processors
- **linux/arm64** - ARM processors (Apple M1/M2, Raspberry Pi, AWS Graviton)

### Image Size

Approximately **350-400 MB** (includes Node.js 20, Gemini CLI, and Go application)

---

## 📖 Additional Documentation

For more detailed guides, see:
- [CONTAINER_AUTHENTICATION.md](CONTAINER_AUTHENTICATION.md) - Detailed authentication guide
- [START_HERE.md](START_HERE.md) - Alternative quick start guide
- [ONE_COMMAND_SETUP.md](ONE_COMMAND_SETUP.md) - One-command setup script

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

---

## 📄 License

This project is open source and available under the MIT License.

---

## ⚠️ Important Notes

### Authentication

1. **✅ DO NOT install Gemini CLI on your computer** - It's already in the container
2. **✅ Authenticate INSIDE the container** - Use `docker exec` command
3. **✅ Always select "1. Login with Google"** when you see the menu
4. **❌ DO NOT use "2. Use Gemini API Key"** - This option will NOT work
5. **❌ DO NOT use "3. Vertex AI"** - This is for enterprise Google Cloud only

### Why This Approach?

**Traditional approach (NOT needed here):**
```bash
❌ npm install -g @google/gemini-cli  # Not needed!
❌ gemini                              # Not needed!
```

**Our approach (correct):**
```bash
✅ docker run ...                      # Start container
✅ docker exec ... gemini              # Authenticate inside container
✅ Select "1. Login with Google"       # Use Google OAuth
```

### Model Selection

6. **Use stable models** for best reliability:
   - ✅ `gemini-2.5-flash-lite`
   - ✅ `gemini-2.5-flash`
   - ✅ `gemini-2.5-pro`
   - ❌ Avoid preview models like `gemini-3-pro`

---

## 🔗 Links

- **Docker Hub**: https://hub.docker.com/r/antiantiops/gemini-wrapper
- **GitHub**: https://github.com/yourusername/gemini-wrapper
- **Gemini CLI**: https://geminicli.com/

---

**Made with ❤️ using Go, Echo, and Google's Gemini CLI**
