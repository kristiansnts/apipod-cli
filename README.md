# apipod-cli

API proxy connector for **Claude Code** (CLI) and **VSCode** (Claude Code extension).

This tool simplifies the process of configuring your local environments to route Claude API requests through a local proxy (default: `http://localhost:8081`).

## Features

- ⚡ **Connect Claude Code**: Automatically configures `~/.claude/settings.json`.
- ⚡ **Connect VSCode**: Configures the Claude Code extension environment variables in VSCode's `settings.json`.
- 🔑 **Token Management**: Easily update your `ANTHROPIC_API_KEY`.
- 🔄 **Reset**: Remove proxy configurations and revert to default settings.

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd apipod-cli

# Install dependencies
npm install

# Link the CLI globally (optional)
npm link
```

## Usage

Run the CLI using:

```bash
apipod
```

Or without linking:

```bash
npm start
```

### Commands

- **Connect to Claude Code (CLI)**: Sets `ANTHROPIC_BASE_URL` to the proxy and saves your API key.
- **Connect to VSCode**: Updates `claude-code.environmentVariables` in your VSCode user settings.
- **Reset**: Remove proxy and API key settings from your configuration files.
- **Change Token**: Update the API key for either or both platforms.

## Configuration Paths

- **Claude Code**: `~/.claude/settings.json`
- **VSCode**:
  - macOS: `~/Library/Application Support/Code/User/settings.json`
  - Windows: `%APPDATA%\Code\User\settings.json`
  - Linux: `~/.config/Code/User/settings.json`

## License

ISC
