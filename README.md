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
# Add: -e OPENAI_API_KEY=sk-local-demo (optional, enables Bearer auth for /v1/*)
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
# Add: -e OPENAI_API_KEY=sk-local-demo (optional, enables Bearer auth for /v1/*)
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

### OpenAI-Compatible Authentication (OPENAI_API_KEY)

Authentication behavior for `/v1/*` depends on container environment:

- If `OPENAI_API_KEY` is **not set**: Bearer token is optional.
- If `OPENAI_API_KEY` **is set**: requests must send `Authorization: Bearer <OPENAI_API_KEY>`.

### Optional model fallback (`FALLBACK_MODEL`)

You can configure fallback models for capacity/rate-limit errors (for example when `gemini-3.1-pro-preview` is exhausted):

- Supports bracket list: `FALLBACK_MODEL=[gemini-2.5-flash,gemini-2.5-flash-lite]`
- Supports comma-separated list: `FALLBACK_MODEL=gemini-2.5-flash,gemini-2.5-flash-lite`
- Retry happens in listed order.
- On successful fallback, logs show the fallback attempt and success model.
- OpenAI-compatible responses return the actual `model` used after fallback.

Run container with OpenAI-compatible API key enabled:

```bash
docker rm -f gemini-wrapper
docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  -e OPENAI_API_KEY=sk-local-demo \
  -e FALLBACK_MODEL=gemini-2.5-flash,gemini-2.5-flash-lite \
  --name gemini-wrapper \
  antiantiops/gemini-wrapper:latest
```

Windows (PowerShell):

```powershell
docker rm -f gemini-wrapper
docker run -d -p 8080:8080 `
  -v ${env:USERPROFILE}\.gemini:/app/.gemini `
  -e OPENAI_API_KEY=sk-local-demo `
  -e FALLBACK_MODEL=gemini-2.5-flash,gemini-2.5-flash-lite `
  --name gemini-wrapper `
  antiantiops/gemini-wrapper:latest
```

Check OpenAI-compatible endpoints:

```bash
# 1) List models (with Bearer token when OPENAI_API_KEY is set)
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-local-demo"

# 2) Chat completion
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-local-demo" \
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

If `OPENAI_API_KEY` is not set, you can remove the `Authorization` header.

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

## Cache Layers

`Ask` uses two cache layers:

- L1: in-memory cache (fast)
- L2: optional disk cache (`bbolt`)

Environment variables:

- `CACHE_ENABLED` (default `true`)
- `CACHE_TTL_SECONDS` (default `1800`)
- `CACHE_MAX_ENTRIES` (default `5000`)
- `CACHE_DEDUPE_ENABLED` (default `true`)
- `CACHE_DISK_ENABLED` (default `true`)
- `CACHE_DISK_PATH` (default `/app/cache/gemini-cache.db`)
- `CACHE_DISK_CLEANUP_INTERVAL_SECONDS` (default `604800`, 7 days)

Behavior:

- On memory miss, the service checks disk cache.
- If disk hit and not expired, it returns cached data and repopulates memory.
- On write, it stores to memory and disk.
- Disk values store: `key`, `answer`, `status_json`, `expires_at_unix`.
- A background cleanup loop removes expired disk keys on the configured interval.

Example:

```bash
docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  -v gemini-wrapper-cache:/app/cache \
  -e CACHE_ENABLED=true \
  -e CACHE_TTL_SECONDS=1800 \
  -e CACHE_MAX_ENTRIES=2000 \
  -e CACHE_DEDUPE_ENABLED=true \
  -e CACHE_DISK_ENABLED=true \
  -e CACHE_DISK_PATH=/app/cache/gemini-cache.db \
  -e CACHE_DISK_CLEANUP_INTERVAL_SECONDS=604800 \
  --name gemini-wrapper \
  antiantiops/gemini-wrapper:latest
```

**Made with ❤️ using Go, Echo, and Google's Gemini CLI**
