# Gemini Wrapper API

[![Docker Hub](https://img.shields.io/docker/v/antiantiops/gemini-wrapper?label=Docker%20Hub&logo=docker)](https://hub.docker.com/r/antiantiops/gemini-wrapper)
[![Docker Pulls](https://img.shields.io/docker/pulls/antiantiops/gemini-wrapper)](https://hub.docker.com/r/antiantiops/gemini-wrapper)
[![Docker Image Size](https://img.shields.io/docker/image-size/antiantiops/gemini-wrapper/latest)](https://hub.docker.com/r/antiantiops/gemini-wrapper)

A Go REST API wrapper for Google's Antigravity CLI (`agy`). Provides a simple HTTP interface (plus OpenAI- and Gemini-compatible endpoints) to interact with Antigravity models.

> Backend: this image ships the **Antigravity CLI (`agy`)**, tested against **agy 1.0.6**. The wrapper invokes `agy --prompt "<question>"` (optionally with `--model`) in headless mode and returns the response.

🐳 **Pre-built Docker images**: https://hub.docker.com/r/antiantiops/gemini-wrapper

---

## 🚨 READ THIS FIRST

### You Do NOT Need to Install the Antigravity CLI on Your Computer!

**❌ WRONG (Traditional Method):**
```bash
# DON'T DO THIS - You don't need to install on localhost!
curl -fsSL https://antigravity.google/cli/install.sh | bash
agy
```

**✅ CORRECT (Our Method):**
```bash
# Just start the container - the Antigravity CLI (agy) is already inside!
docker run -d -p 8080:8080 -v ~/.gemini:/app/.gemini --name gemini-wrapper antiantiops/gemini-wrapper:latest

# Then authenticate INSIDE the container
docker exec -it gemini-wrapper sh -c 'export HOME=/app && export ANTIGRAVITY_CONFIG_DIR=/app/.gemini && cd /app && agy'
```

**Why our method is better:**
- ✅ No Node.js installation on your computer
- ✅ No Antigravity CLI installation on your computer
- ✅ Everything isolated in Docker
- ✅ Only Docker required

---

## 🎯 How It Works

**You do NOT need to install anything on your computer except Docker!**

```
### Step 1:
agy

###Step 2: Choose Google Oauth

     ▄▀▀▄
    ▀▀▀▀▀▀
   ▀▀▀▀▀▀▀▀
  ▄▀▀    ▀▀▄
 ▄▀▀      ▀▀▄

 Welcome to the Antigravity CLI. You are currently not signed in.

 Select login method:
 > 1. Google OAuth
   2. Use a Google Cloud project

 [Use arrow keys to navigate, Enter to select]


### Step 3: get out of the container

                           ↓ Mount
┌─────────────────────────────────────────────────────────────┐
│                   DOCKER CONTAINER                           │
│                                                              │
│  • Antigravity CLI (agy) pre-installed ✅                    │
│  • Node.js pre-installed ✅                                  │
│  • Go application pre-installed ✅                           │
│                                                              │
│  3. You run: agy (inside container)                         │
│  4. Complete the Antigravity sign-in flow                   │
│  5. Credentials saved to /app/.gemini                       │
│                                                              │
└──────────────────────────────────────────────────────────────┘
                           ↓ Mount (bidirectional)
┌─────────────────────────────────────────────────────────────┐
│                    YOUR COMPUTER (Host)                      │
│                                                              │
│  6. Credentials appear in: ~/.gemini ✅                      │
│  7. Container can now access Antigravity ✅                  │
│  8. Your REST API is ready! ✅                               │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Key Points:**
- ✅ **No localhost installation** - Everything runs in Docker
- ✅ **Authenticate in container** - Not on your computer
- ✅ **Credentials shared via mount** - Saved to both container and host

---

## ⚡ Quick Start (3 Steps)

### Prerequisites

**✅ Only Docker is required!**

**❌ You do NOT need to:**
- Install Node.js on your computer
- Install the Antigravity CLI on your computer

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

Run this command to enter the container and start the Antigravity sign-in flow:

```bash
docker exec -it gemini-wrapper sh -c 'export HOME=/app && export ANTIGRAVITY_CONFIG_DIR=/app/.gemini && cd /app && agy'
```

Follow the on-screen prompts from `agy` to sign in. When the CLI shows a URL:

1. **Copy the entire URL** it prints
2. **Open it in your browser** (on your host computer)
3. **Sign in** and grant permissions
4. **Copy the authorization code** the browser shows
5. **Go back to the container terminal** and paste the code
6. **Press Enter** until the CLI confirms you are signed in

> Note: the exact sign-in prompts depend on your `agy` version. This image is tested with **agy 1.0.6** — run `agy --help` and `agy install` inside the container if you need to (re)configure environment paths.

**What happened:**
- You authenticated inside the container
- Credentials were saved to `/app/.gemini` (inside container)
- Because `/app/.gemini` is mounted to `~/.gemini` (on your computer)
- The credentials are now available on both your computer AND in the container

Verify the CLI works headlessly:
```bash
docker exec -it gemini-wrapper sh -c 'export HOME=/app && export ANTIGRAVITY_CONFIG_DIR=/app/.gemini && agy --prompt "What model are you? One sentence."'
```

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

**✅ That's it! Your Antigravity API is ready to use.**

---

## 📝 Quick Summary

### What You Just Did:

1. ✅ Created empty folder: `~/.gemini` on your computer
2. ✅ Started Docker container with folder mounted
3. ✅ Ran `agy` command **INSIDE the container** (not on your computer!)
4. ✅ Completed the Antigravity sign-in flow
5. ✅ Credentials saved to container's `/app/.gemini`
6. ✅ Credentials automatically appear in your `~/.gemini` (via mount)
7. ✅ Restarted container
8. ✅ API is now working!

### What You Did NOT Do:

- ❌ Install Node.js on your computer
- ❌ Install the Antigravity CLI on your computer (`agy`)
- ❌ Run `agy` command on your computer

### Why This Approach?

**Everything happens inside Docker:**
- The Antigravity CLI (`agy`) is already installed in the container
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
    "model": "Gemini 3.5 Flash (Medium)"
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
curl -X POST "http://localhost:8080/v1beta/models/Gemini%203.5%20Flash%20(Medium)" \
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
  "model": "Gemini 3.5 Flash (Medium)",
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

This backend exposes OpenAI-compatible endpoints and forwards generation traffic to the Antigravity CLI (`agy`):

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/completions`
- `POST /v1/responses`

### OpenAI-Compatible Authentication (OPENAI_API_KEY)

Authentication behavior for `/v1/*` depends on container environment:

- If `OPENAI_API_KEY` is **not set**: Bearer token is optional.
- If `OPENAI_API_KEY` **is set**: requests must send `Authorization: Bearer <OPENAI_API_KEY>`.

### Optional model fallback (`FALLBACK_MODEL`)

You can configure fallback models for capacity/rate-limit errors (for example when a Pro model is exhausted):

- Supports bracket list: `FALLBACK_MODEL=[Gemini 3.5 Flash (Medium),Gemini 3.5 Flash (Low)]`
- Supports comma-separated list: `FALLBACK_MODEL=Gemini 3.5 Flash (Medium),Gemini 3.5 Flash (Low)`
- Retry happens in listed order.
- On successful fallback, logs show the fallback attempt and success model.
- OpenAI-compatible responses return the actual `model` used after fallback.

Run container with OpenAI-compatible API key enabled:

```bash
docker rm -f gemini-wrapper
docker run -d -p 8080:8080 \
  -v ~/.gemini:/app/.gemini \
  -e OPENAI_API_KEY=sk-local-demo \
  -e "FALLBACK_MODEL=Gemini 3.5 Flash (Medium),Gemini 3.5 Flash (Low)" \
  --name gemini-wrapper \
  antiantiops/gemini-wrapper:latest
```

Windows (PowerShell):

```powershell
docker rm -f gemini-wrapper
docker run -d -p 8080:8080 `
  -v ${env:USERPROFILE}\.gemini:/app/.gemini `
  -e OPENAI_API_KEY=sk-local-demo `
  -e "FALLBACK_MODEL=Gemini 3.5 Flash (Medium),Gemini 3.5 Flash (Low)" `
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
    "model": "Gemini 3.5 Flash (Medium)",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

If `OPENAI_API_KEY` is not set, you can remove the `Authorization` header.

---

## 🎯 Available Models

These are the exact model names accepted by `agy 1.0.6` (run `agy models` inside the container to confirm for your version):

| Model (`agy` display name) | Notes |
|----------------------------|-------|
| `Gemini 3.5 Flash (Medium)` | Balanced Flash tier |
| `Gemini 3.5 Flash (High)` | Higher-effort Flash |
| `Gemini 3.5 Flash (Low)` | Fastest / cheapest Flash |
| `Gemini 3.1 Pro (Low)` | Pro, lower effort |
| `Gemini 3.1 Pro (High)` | Pro, highest quality |
| `Claude Sonnet 4.6 (Thinking)` | Anthropic Sonnet |
| `Claude Opus 4.6 (Thinking)` | Anthropic Opus |
| `GPT-OSS 120B (Medium)` | Open-weight GPT-OSS |

> ⚠️ `agy` silently falls back to its default model when given an unknown name. To avoid surprises, the wrapper resolves a set of convenience **aliases** to the exact display names above, and forwards anything unrecognized as-is (with a warning in the logs).

### Model aliases

| Alias | Resolves to |
|-------|-------------|
| `gemini-3.5-flash`, `gemini-flash` | `Gemini 3.5 Flash (Medium)` |
| `gemini-3.1-pro`, `gemini-pro` | `Gemini 3.1 Pro (High)` |
| `claude-sonnet-4.6`, `claude-sonnet` | `Claude Sonnet 4.6 (Thinking)` |
| `claude-opus-4.6`, `claude-opus` | `Claude Opus 4.6 (Thinking)` |
| `gpt-oss-120b`, `gpt-oss` | `GPT-OSS 120B (Medium)` |

If you omit `model` (or send `antigravity-default`), the wrapper does **not** pass `--model` and lets `agy` use its session default.

**Examples:**

```bash
# Use a Flash tier
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Hi!", "model": "Gemini 3.5 Flash (Low)"}'

# Default (no model) - agy picks its session default
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Explain Docker"}'

# High quality via alias
curl -X POST http://localhost:8080/api/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Write a research outline on AI", "model": "claude-opus"}'
```

---

## Antigravity CLI Configuration

The wrapper invokes the Antigravity CLI in headless mode. These environment variables control how it is called:

- `ANTIGRAVITY_CLI_COMMAND` (default `agy`) — the CLI binary to execute.
- `ANTIGRAVITY_CONFIG_DIR` (default `/app/.gemini`) — config/credentials dir, exported as `ANTIGRAVITY_CONFIG_DIR` to the CLI. This is where `agy` stores its OAuth credentials (`oauth_creds.json`, `antigravity-cli/`, etc.), so it must be the mounted volume.
- `ANTIGRAVITY_HOME` (default `/app`) — value used for `HOME` and `XDG_CONFIG_HOME` when invoking the CLI. Set this when running outside Docker.
- `ANTIGRAVITY_CLI_TIMEOUT_SECONDS` (default `300`) — hard timeout for a single CLI call; prevents a hung process from blocking requests.
- `ANTIGRAVITY_SKIP_PERMISSIONS` (default `false`) — when truthy, passes `--dangerously-skip-permissions` so tool-permission prompts are auto-approved (avoids blocking on stdin). Security sensitive: only enable for trusted, headless use.

> Notes:
> - `agy 1.0.6` does not support an `--output-format` flag; the wrapper parses plain-text output (and JSON when present).
> - The wrapper calls `agy --prompt "<question>"` and adds `--model "<resolved name>"` only when a recognized model/alias is supplied.

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

**Made with ❤️ using Go, Echo, and Google's Antigravity CLI (`agy`)**
