# apipod-cli

An agentic coding assistant for your terminal, powered by [Apipod](https://apipod.net).

Like Claude Code, Copilot CLI, or OpenCode — but connected to your Apipod proxy with multi-provider AI routing.

## Features

- 🤖 **Full agentic coding** — reads/writes files, runs bash, searches code
- 📡 **Streaming responses** — real-time SSE streaming from Anthropic Messages API
- 🔧 **Client-side tools** — Bash, Read, Write, Edit, MultiEdit, Glob, Grep
- 🔐 **Device auth login** — `apipod-cli login` opens browser for secure authentication
- 🎯 **Smart model routing** — uses Apipod proxy for multi-provider orchestration
- ⚡ **Single binary** — Go binary, no runtime dependencies

## Installation

### From source

```bash
go install github.com/rpay/apipod-cli/cmd/apipod-cli@latest
```

### Build locally

```bash
git clone https://github.com/rpay/apipod-cli.git
cd apipod-cli
go build -o apipod-cli ./cmd/apipod-cli/
```

## Quick Start

```bash
# Login to your Apipod account
apipod-cli login

# Start interactive session
apipod-cli

# Or send a single prompt
apipod-cli "explain this codebase"

# Use a specific model
apipod-cli --model claude-sonnet-4-20250514
```

## Commands

| Command | Description |
|---------|-------------|
| `apipod-cli` | Start interactive REPL |
| `apipod-cli "prompt"` | Send a single prompt |
| `apipod-cli login` | Authenticate via browser |
| `apipod-cli logout` | Remove saved credentials |
| `apipod-cli whoami` | Show current user info |
| `apipod-cli --model MODEL` | Use a specific model |
| `apipod-cli --help` | Show help |

## Slash Commands (in interactive mode)

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/clear` | Clear conversation history |
| `/model [name]` | Show or change model |
| `/compact` | Clear context |
| `/whoami` | Show current user |
| `/quit` | Exit |

## Configuration

Config is stored at `~/.apipod/config.json`:

```json
{
  "base_url": "https://api.apipod.net",
  "api_key": "apk_...",
  "model": "claude-sonnet-4-20250514",
  "username": "your-name",
  "plan": "pro"
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `APIPOD_BASE_URL` | API base URL (overrides config) |
| `APIPOD_API_KEY` | API key (overrides config) |
| `APIPOD_MODEL` | Default model (overrides config) |

## License

MIT
