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
2. **Environment** — Shared API keys are loaded from `~/orch/.env` into the session
3. **Launch** — A tmux session is created and Claude Code is started inside it
4. **Trust** — The "trust this folder" prompt is auto-confirmed
5. **Prompt** — Your initial prompt is sent once Claude is ready
6. **Sync** — The conversation appears in the Claude mobile app for you to continue
7. **Heartbeat** — Sessions update a heartbeat file every ~5 minutes with their status
8. **Report** — Sessions drop reports to `~/orch/inbox/` for the orchestrator to read

## Commands

| Command | Description |
|---|---|
| `session orch` | Start the orchestrator session in `~/orch` |
| `session start <org/repo>` | Clone repo (if needed) and launch Claude Code in tmux |
| `session list` | List active sessions with heartbeat status |
| `session status <name>` | Peek at what a session is currently doing |
| `session attach <name>` | Attach to a running session's terminal |
| `session send <name> <msg>` | Send text input to a running session |
| `session inbox` | List reports from spawned sessions |
| `session inbox --read` | Show full report contents |
| `session inbox --clear` | Clear all reports after reading |
| `session stop <name>` | Kill a session |
| `session stop --all` | Kill all sessions |
| `session watchdog` | Check if orch is alive, restart if not (for cron/launchd) |

## Usage

### The Orchestrator

The orchestrator is the central Claude Code session that manages everything else. It runs in `~/orch`.

```bash
# Start the orchestrator
session orch

# Start with a custom prompt
session orch -p "check on all projects and report status"

# Resume the previous conversation
session orch --resume

# Run autonomously
session orch --dangerously-skip-permissions
```

### Watchdog

The watchdog ensures the orchestrator stays running. Set it up with macOS launchd:

```xml
<!-- ~/Library/LaunchAgents/dev.thunk.session-watchdog.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.thunk.session-watchdog</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/session</string>
        <string>watchdog</string>
    </array>
    <key>StartInterval</key>
    <integer>300</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>HOME</key>
        <string>/Users/you</string>
    </dict>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/dev.thunk.session-watchdog.plist
```

Or with cron: `*/5 * * * * /usr/local/bin/session watchdog`

### Start a project session

```bash
# Start Claude Code in a project
session start kathunk/tidy

# Start with an initial prompt
session start kathunk/tidy -p "fix the login bug in AuthController"

# Resume the most recent conversation
session start kathunk/tidy --resume

# Full autonomy
session start kathunk/tidy --dangerously-skip-permissions -p "fix and deploy"
```

If no prompt is given, Claude is sent a default message asking it to read `CLAUDE.md` and report ready.

### Repo resolution

| Input | Resolved to |
|---|---|
| `kathunk/tidy` | `github.com/kathunk/tidy` → `~/src/tidy` |
| `my-project` | `github.com/DanielCoulbourne/my-project` → `~/src/my-project` |
| `https://github.com/org/repo` | `github.com/org/repo` → `~/src/repo` |
| `git@github.com:org/repo.git` | `github.com/org/repo` → `~/src/repo` |

### Monitor sessions

```bash
# See what's running with heartbeat status
session list
# SESSION              STATUS       LAST ACTIVE    CURRENT TASK
# orch                 working      2m ago         reviewing PR #218
# tidy                 idle         15m ago        waiting for input

# Peek at a session's current screen
session status tidy

# Check reports from sessions
session inbox
session inbox --read
session inbox --read --session tidy

# Clear reports after reading
session inbox --clear
```

### Interact with sessions

```bash
# Send a follow-up message
session send tidy "now run the test suite"

# Attach to interact directly in the terminal
session attach tidy

# Stop a session
session stop tidy

# Stop everything
session stop --all
```

## Heartbeat

Sessions are instructed to write a heartbeat file every ~5 minutes during active work:

```
~/orch/inbox/heartbeat-tidy.md
```

```markdown
# tidy: Heartbeat
Last active: 2026-03-08T15:49:14Z
Status: working
Current task: implementing OAuth PKCE flow
```

`session list` reads these to show live status. Status values: `working`, `idle`, `blocked`, `completed`.

## Shared Tooling

Sessions automatically inherit environment variables from `~/orch/.env`. This is where you put shared API keys (e.g., `LINEAR_API_KEY`) so all spawned sessions have access to the same tooling.

Any CLI tools installed globally (e.g., `linear`, `gh`) are available to all sessions.

## Reporting

Spawned sessions are instructed (via `--append-system-prompt`) to drop reports to `~/orch/inbox/` when they complete tasks, hit blockers, or have questions. Reports are markdown files named with a timestamp and slug:

```
~/orch/inbox/20260308-153200-auth-fix-complete.md
```

The orchestrator reads these with `session inbox`.

## Running from inside Claude Code

This tool is designed to be called from within a Claude Code session (e.g., an orchestration hub). It automatically handles the `CLAUDECODE` environment variable to avoid nested session detection.

## Configuration

| Setting | Default | Location |
|---|---|---|
| Project directory | `~/src/` | `cmd/root.go` → `srcDir` |
| Orchestrator directory | `~/orch/` | `cmd/root.go` → `orchDir` |
| Inbox directory | `~/orch/inbox/` | `cmd/root.go` → `inboxDir` |
| Shared env vars | `~/orch/.env` | Sourced before launching claude |
| Watchdog interval | 5 minutes | launchd plist `StartInterval` |
