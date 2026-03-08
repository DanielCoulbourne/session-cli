package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var orchCmd = &cobra.Command{
	Use:   "orch",
	Short: "Start the orchestrator Claude Code session",
	Long: `Launch Claude Code in a tmux session named "orch" rooted at ~/orch.

This is the main orchestrator session that manages all other sessions.

Examples:
  session orch
  session orch -p "Check inbox and prioritize tasks"
  session orch --resume
  session orch --dangerously-skip-permissions`,
	Args: cobra.NoArgs,
	RunE: runOrch,
}

func init() {
	orchCmd.Flags().StringP("prompt", "p", "", "Initial prompt to send to Claude Code")
	orchCmd.Flags().Bool("resume", false, "Resume the most recent Claude Code conversation")
	orchCmd.Flags().Bool("dangerously-skip-permissions", false, "Skip all permission checks in the spawned session")
	rootCmd.AddCommand(orchCmd)
}

func runOrch(cmd *cobra.Command, args []string) error {
	prompt, _ := cmd.Flags().GetString("prompt")
	resume, _ := cmd.Flags().GetBool("resume")
	skipPerms, _ := cmd.Flags().GetBool("dangerously-skip-permissions")

	sessionName := "orch"

	// Check if tmux session already exists
	if sessionExists(sessionName) {
		// Check if claude is actually running in it
		out, _ := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p").Output()
		pane := string(out)
		claudeRunning := strings.Contains(pane, "claude") || strings.Contains(pane, "Claude") ||
			strings.Contains(pane, "Tips:") || strings.Contains(pane, "/help") ||
			strings.Contains(pane, "❯") || strings.Contains(pane, "What can I")
		if claudeRunning {
			fmt.Printf("Session %q already exists with Claude running. Use 'session attach %s' to connect.\n", sessionName, sessionName)
			return nil
		}
		// Session exists but Claude isn't running — relaunch Claude in it
		fmt.Printf("Session %q exists but Claude is not running. Relaunching...\n", sessionName)
		claudeArgs := []string{}
		if resume {
			claudeArgs = append(claudeArgs, "--resume")
		}
		if skipPerms {
			claudeArgs = append(claudeArgs, "--dangerously-skip-permissions")
		}
		relaunchCmd := "unset CLAUDECODE && claude " + strings.Join(claudeArgs, " ")
		sendClaude := exec.Command("tmux", "send-keys", "-t", sessionName, relaunchCmd, "Enter")
		if err := sendClaude.Run(); err != nil {
			return fmt.Errorf("relaunching claude: %w", err)
		}
		if prompt == "" && !resume {
			prompt = "You are the orchestrator. Read CLAUDE.md and let me know you're ready."
		}
		if prompt != "" {
			waitAndSendPrompt(sessionName, prompt)
		}
		fmt.Printf("Relaunched orchestrator in session %q\n", sessionName)
		fmt.Printf("Attach with: session attach %s\n", sessionName)
		return nil
	}

	// Ensure inbox directory exists
	os.MkdirAll(inboxDir, 0755)

	// Build the claude command
	claudeArgs := []string{}
	if resume {
		claudeArgs = append(claudeArgs, "--resume")
	}
	if skipPerms {
		claudeArgs = append(claudeArgs, "--dangerously-skip-permissions")
	}

	claudeCmd := "claude " + strings.Join(claudeArgs, " ")

	if prompt == "" {
		if resume {
			prompt = "Continue where you left off. Check inbox and let me know what's going on."
		} else {
			prompt = "You are the orchestrator. Read CLAUDE.md and let me know you're ready."
		}
	}

	// Unset CLAUDECODE so the spawned session doesn't inherit it
	envSetup := "unset CLAUDECODE"

	// Create tmux session rooted in ~/orch
	tmuxCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", orchDir)
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
		waitAndSendPrompt(sessionName, prompt)
	}

	fmt.Printf("Started orchestrator session %q in %s\n", sessionName, orchDir)
	fmt.Printf("Attach with: session attach %s\n", sessionName)
	return nil
}

func waitAndSendPrompt(sessionName, prompt string) {
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
					exec.Command("tmux", "send-keys", "-t", sessionName, prompt, "Enter").Run()
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
