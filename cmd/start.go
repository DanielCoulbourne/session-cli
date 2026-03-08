package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [org/repo]",
	Short: "Start a Claude Code session for a project",
	Long: `Clone the repo (if not already at ~/src/{repo}) and launch Claude Code
in a new tmux session with access to shared tooling and a reporting channel.

The repo argument can be:
  - org/repo (e.g. kathunk/tidy)
  - A full GitHub URL
  - Just a repo name (assumes DanielCoulbourne org)

Examples:
  session start kathunk/tidy
  session start kathunk/tidy -p "fix the login bug"
  session start kathunk/tidy --resume
  session start my-project`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

func init() {
	startCmd.Flags().StringP("prompt", "p", "", "Initial prompt to send to Claude Code")
	startCmd.Flags().Bool("resume", false, "Resume the most recent Claude Code conversation")
	startCmd.Flags().Bool("dangerously-skip-permissions", false, "Skip all permission checks in the spawned session")
	startCmd.Flags().StringSlice("env", nil, "Additional .env files to source (default: ~/orch/.env)")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	prompt, _ := cmd.Flags().GetString("prompt")
	resume, _ := cmd.Flags().GetBool("resume")
	skipPerms, _ := cmd.Flags().GetBool("dangerously-skip-permissions")
	extraEnvFiles, _ := cmd.Flags().GetStringSlice("env")

	repo := args[0]

	// Parse org/repo from various formats
	org, repoName := parseRepo(repo)
	fullRepo := org + "/" + repoName
	sessionName := sanitizeSessionName(repoName)
	repoDir := filepath.Join(srcDir, repoName)

	// Check if tmux session already exists
	if sessionExists(sessionName) {
		fmt.Printf("Session %q already exists. Use 'session attach %s' to connect.\n", sessionName, sessionName)
		return nil
	}

	// Ensure inbox directory exists
	os.MkdirAll(inboxDir, 0755)

	// Clone if needed
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		fmt.Printf("Cloning %s into %s...\n", fullRepo, repoDir)
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			return fmt.Errorf("creating src directory: %w", err)
		}
		cloneURL := fmt.Sprintf("https://github.com/%s.git", fullRepo)
		cloneCmd := exec.Command("git", "clone", cloneURL, repoDir)
		cloneCmd.Stdout = os.Stdout
		cloneCmd.Stderr = os.Stderr
		if err := cloneCmd.Run(); err != nil {
			return fmt.Errorf("cloning repo: %w", err)
		}
	} else {
		fmt.Printf("Repo already exists at %s\n", repoDir)
	}

	// Enable Entire if available and not already enabled
	if _, err := exec.LookPath("entire"); err == nil {
		entireSettings := filepath.Join(repoDir, ".entire", "settings.json")
		if _, err := os.Stat(entireSettings); os.IsNotExist(err) {
			fmt.Printf("Enabling Entire for %s...\n", repoName)
			enableCmd := exec.Command("entire", "enable", "--agent", "claude-code")
			enableCmd.Dir = repoDir
			enableCmd.Env = append(os.Environ(), "ACCESSIBLE=1")
			enableCmd.Stdout = os.Stdout
			enableCmd.Stderr = os.Stderr
			if err := enableCmd.Run(); err != nil {
				fmt.Printf("Warning: failed to enable Entire: %v\n", err)
			}
		}
	}

	// Build the claude command with system prompt for reporting
	claudeArgs := []string{}
	if resume {
		claudeArgs = append(claudeArgs, "--resume")
	}
	if skipPerms {
		claudeArgs = append(claudeArgs, "--dangerously-skip-permissions")
	}

	// Inject system prompt with tooling and reporting instructions
	systemPrompt := buildSystemPrompt(sessionName)
	claudeArgs = append(claudeArgs, "--append-system-prompt", shelljoin([]string{systemPrompt}))

	claudeCmd := "claude " + strings.Join(claudeArgs, " ")

	if prompt == "" && !resume {
		prompt = fmt.Sprintf("You are working on the %s project. Read CLAUDE.md if it exists and let me know you're ready.", repoName)
	}

	// Build env setup command — source orch .env + any extra env files
	envSetup := "unset CLAUDECODE"
	orchEnv := filepath.Join(orchDir, ".env")
	if _, err := os.Stat(orchEnv); err == nil {
		envSetup += fmt.Sprintf(" && set -a && source %s && set +a", orchEnv)
	}
	for _, envFile := range extraEnvFiles {
		envSetup += fmt.Sprintf(" && set -a && source %s && set +a", envFile)
	}

	// Create tmux session with a shell
	tmuxCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", repoDir)
	tmuxCmd.Stdout = os.Stdout
	tmuxCmd.Stderr = os.Stderr
	if err := tmuxCmd.Run(); err != nil {
		return fmt.Errorf("creating tmux session: %w", err)
	}

	// Source env vars and launch claude
	time.Sleep(500 * time.Millisecond)
	launchCmd := fmt.Sprintf("%s && %s", envSetup, claudeCmd)
	sendClaude := exec.Command("tmux", "send-keys", "-t", sessionName, launchCmd, "Enter")
	if err := sendClaude.Run(); err != nil {
		return fmt.Errorf("launching claude: %w", err)
	}

	if prompt != "" {
		fmt.Printf("Waiting for Claude to start...")
		promptSent := false
		for i := 0; i < 60; i++ {
			time.Sleep(1 * time.Second)
			out, _ := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p").Output()
			pane := string(out)

			// Handle trust prompt — press Enter to confirm
			if strings.Contains(pane, "Yes, I trust this folder") {
				exec.Command("tmux", "send-keys", "-t", sessionName, "Enter").Run()
				fmt.Print("(trusted).")
				continue
			}

			// Claude is ready when it shows the input prompt
			lines := strings.Split(strings.TrimSpace(pane), "\n")
			if len(lines) > 0 {
				lastLine := strings.TrimSpace(lines[len(lines)-1])
				if lastLine == ">" || lastLine == "❯" || lastLine == "" {
					if strings.Contains(pane, "Tips:") || strings.Contains(pane, "/help") || strings.Contains(pane, "What can I") {
						fmt.Println(" ready!")
						time.Sleep(500 * time.Millisecond)
						sendPrompt := exec.Command("tmux", "send-keys", "-t", sessionName, prompt, "Enter")
						if err := sendPrompt.Run(); err != nil {
							return fmt.Errorf("sending prompt: %w", err)
						}
						promptSent = true
						break
					}
				}
			}
			fmt.Print(".")
		}
		if !promptSent {
			fmt.Println(" timed out, sending prompt anyway.")
			time.Sleep(500 * time.Millisecond)
			exec.Command("tmux", "send-keys", "-t", sessionName, prompt, "Enter").Run()
		}
	}

	fmt.Printf("Started session %q in %s\n", sessionName, repoDir)
	fmt.Printf("Attach with: session attach %s\n", sessionName)
	return nil
}

func buildSystemPrompt(sessionName string) string {
	return fmt.Sprintf(`## Orchestrator Integration

You were launched by the orchestrator session. You have access to shared tooling and a reporting channel.

### Shared Tooling
- The 'linear' CLI is available at /usr/local/bin/linear for project management tasks.
  Run 'linear --help' for usage. Use '-o json' for machine-readable output.
  API keys are pre-configured in your environment.

### Reporting Back
When you complete a task, hit a blocker, or have something the orchestrator needs to know,
write a report file to the inbox:

  Write a file to: %s/{timestamp}-{brief-slug}.md

Format:
  # {Session}: {Brief Title}
  Status: completed | blocked | update | question

  {Details}

Use the Bash tool to write reports:
  echo '...' > %s/$(date +%%Y%%m%%d-%%H%%M%%S)-{slug}.md

Only report when you have something meaningful — don't spam the inbox.
The orchestrator will check it periodically.

### Heartbeat
While actively working, update a heartbeat file approximately every 5 minutes so the orchestrator
can monitor your status at a glance. Also update it immediately whenever your task status changes
(e.g. you become blocked, complete a task, start idling, etc.).

Write/overwrite (not append) this file:
  %s/heartbeat-%s.md

Format (exactly):
  # %s: Heartbeat
  Last active: {ISO 8601 timestamp, e.g. 2026-03-08T14:30:00Z}
  Status: working | idle | blocked | completed
  Current task: {brief one-line description of what you're doing}

Use the Bash tool to write it:
  cat > %s/heartbeat-%s.md << 'HEARTBEAT'
  # %s: Heartbeat
  Last active: $(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)
  Status: working
  Current task: implementing login fix
  HEARTBEAT

### Session Identity
Your session name is: %s
`, inboxDir, inboxDir,
		inboxDir, sessionName,
		sessionName,
		inboxDir, sessionName,
		sessionName,
		sessionName)
}

func parseRepo(repo string) (org, name string) {
	// Strip GitHub URL prefix
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "git@github.com:")
	repo = strings.TrimSuffix(repo, ".git")

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "DanielCoulbourne", parts[0]
}

func sanitizeSessionName(name string) string {
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, ":", "-")
	return name
}

func sessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

func shelljoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n\"'") {
			quoted[i] = "'" + strings.ReplaceAll(a, "'", "'\\''") + "'"
		} else {
			quoted[i] = a
		}
	}
	return strings.Join(quoted, " ")
}
