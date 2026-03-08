# session-cli

A CLI tool for managing [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions across multiple projects using tmux. Start Claude Code in any GitHub repo, send it prompts, and pick up the conversation on the Claude mobile app — all without touching a keyboard.

Built for orchestrating AI-assisted development across many projects at once.

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [tmux](https://github.com/tmux/tmux)
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) (`claude`)
- [GitHub CLI](https://cli.github.com/) (`gh`) — for authentication (HTTPS cloning)

```bash
brew install tmux gh
```

## Install

```bash
git clone https://github.com/DanielCoulbourne/session-cli.git
cd session-cli
go build -o session .
cp session /usr/local/bin/
```

## How It Works

1. **Clone** — Repos are cloned to `~/src/{reponame}` (skipped if already present)
2. **Launch** — A tmux session is created and Claude Code is started inside it
3. **Trust** — The "trust this folder" prompt is auto-confirmed
4. **Prompt** — Your initial prompt is sent once Claude is ready
5. **Sync** — The conversation appears in the Claude mobile app for you to continue

This means you can spawn sessions from one place (e.g., an orchestration hub or the mobile app) and pick them up anywhere.

## Commands

| Command | Description |
|---|---|
| `session start <org/repo>` | Clone repo (if needed) and launch Claude Code in tmux |
| `session list` | List active sessions |
| `session attach <name>` | Attach to a running session's terminal |
| `session send <name> <msg>` | Send text input to a running session |
| `session stop <name>` | Kill a session |
| `session stop --all` | Kill all sessions |

## Usage

### Start a session

```bash
# Start Claude Code in a project
session start kathunk/tidy

# Start with an initial prompt
session start kathunk/tidy -p "fix the login bug in AuthController"

# Resume the most recent conversation
session start kathunk/tidy --resume
```

If no prompt is given, Claude is sent a default message asking it to read `CLAUDE.md` and report ready.

### Repo resolution

The repo argument is flexible:

| Input | Resolved to |
|---|---|
| `kathunk/tidy` | `github.com/kathunk/tidy` → `~/src/tidy` |
| `my-project` | `github.com/DanielCoulbourne/my-project` → `~/src/my-project` |
| `https://github.com/org/repo` | `github.com/org/repo` → `~/src/repo` |
| `git@github.com:org/repo.git` | `github.com/org/repo` → `~/src/repo` |

### Manage sessions

```bash
# See what's running
session list

# Send a follow-up message
session send tidy "now run the test suite"

# Attach to interact directly in the terminal
session attach tidy

# Stop a session
session stop tidy

# Stop everything
session stop --all
```

## Running from inside Claude Code

This tool is designed to be called from within a Claude Code session (e.g., an orchestration hub). It automatically handles the `CLAUDECODE` environment variable to avoid nested session detection.

## Configuration

Projects are cloned to `~/src/` by default. This is currently hardcoded in `cmd/root.go` — change `srcDir` if you prefer a different location.
