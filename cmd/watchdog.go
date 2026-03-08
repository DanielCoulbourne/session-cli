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

var watchdogCmd = &cobra.Command{
	Use:   "watchdog",
	Short: "Check if the orch session is alive and restart it if not",
	Long: `Designed to be run from cron. Checks if the "orch" tmux session exists.
If it does, exits silently. If not, restarts it with --dangerously-skip-permissions --resume.

Logs restarts to ~/orch/inbox/watchdog.log with timestamps.

Example crontab entry:
  */5 * * * * /usr/local/bin/session watchdog`,
	Args: cobra.NoArgs,
	RunE: runWatchdog,
}

func init() {
	rootCmd.AddCommand(watchdogCmd)
}

func runWatchdog(cmd *cobra.Command, args []string) error {
	sessionName := "orch"

	// If session exists, exit silently
	if sessionExists(sessionName) {
		return nil
	}

	// Log the restart
	logFile := filepath.Join(inboxDir, "watchdog.log")
	os.MkdirAll(inboxDir, 0755)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] Orch session not found, restarting...\n", timestamp)

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening watchdog log: %w", err)
	}
	f.WriteString(logEntry)
	f.Close()

	// Start the orch session with --dangerously-skip-permissions --resume
	claudeCmd := "claude --resume --dangerously-skip-permissions"
	envSetup := "unset CLAUDECODE"
	prompt := "You are the orchestrator. Read CLAUDE.md and let me know you're ready."

	// Create tmux session rooted in ~/orch
	tmuxCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", orchDir)
	tmuxCmd.Stdout = os.Stdout
	tmuxCmd.Stderr = os.Stderr
	if err := tmuxCmd.Run(); err != nil {
		logFailure(logFile, timestamp, "creating tmux session", err)
		return fmt.Errorf("creating tmux session: %w", err)
	}

	// Launch claude
	time.Sleep(500 * time.Millisecond)
	launchStr := fmt.Sprintf("%s && %s", envSetup, claudeCmd)
	sendClaude := exec.Command("tmux", "send-keys", "-t", sessionName, launchStr, "Enter")
	if err := sendClaude.Run(); err != nil {
		logFailure(logFile, timestamp, "launching claude", err)
		return fmt.Errorf("launching claude: %w", err)
	}

	// Wait for Claude to be ready and send prompt
	for i := 0; i < 60; i++ {
		time.Sleep(1 * time.Second)
		out, _ := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p").Output()
		pane := string(out)

		// Handle trust prompt
		if strings.Contains(pane, "Yes, I trust this folder") {
			exec.Command("tmux", "send-keys", "-t", sessionName, "Enter").Run()
			continue
		}

		// Claude is ready when it shows the input prompt
		lines := strings.Split(strings.TrimSpace(pane), "\n")
		if len(lines) > 0 {
			lastLine := strings.TrimSpace(lines[len(lines)-1])
			if lastLine == ">" || lastLine == "❯" || lastLine == "" {
				if strings.Contains(pane, "Tips:") || strings.Contains(pane, "/help") || strings.Contains(pane, "What can I") {
					time.Sleep(500 * time.Millisecond)
					exec.Command("tmux", "send-keys", "-t", sessionName, prompt, "Enter").Run()
					logSuccess(logFile, timestamp)
					return nil
				}
			}
		}
	}

	// Timed out waiting, send prompt anyway
	time.Sleep(500 * time.Millisecond)
	exec.Command("tmux", "send-keys", "-t", sessionName, prompt, "Enter").Run()
	logSuccess(logFile, timestamp)
	return nil
}

func logFailure(logFile, timestamp, step string, err error) {
	f, ferr := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if ferr != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("[%s] Failed %s: %v\n", timestamp, step, err))
}

func logSuccess(logFile, timestamp string) {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("[%s] Orch session restarted successfully.\n", timestamp))
}
