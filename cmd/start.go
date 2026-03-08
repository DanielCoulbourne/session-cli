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
in a new tmux session.

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
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	prompt, _ := cmd.Flags().GetString("prompt")
	resume, _ := cmd.Flags().GetBool("resume")

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

	// Build the claude command
	// Claude Code runs interactively by default (no args).
	// We start it interactively in tmux, then send the prompt as keystrokes.
	claudeArgs := ""
	if resume {
		claudeArgs = "--resume"
	}

	claudeCmd := "claude"
	if claudeArgs != "" {
		claudeCmd = "claude " + claudeArgs
	}

	if prompt == "" && !resume {
		prompt = fmt.Sprintf("You are working on the %s project. Read CLAUDE.md if it exists and let me know you're ready.", repoName)
	}

	// Create tmux session with a shell first, then launch claude
	tmuxCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", repoDir)
	tmuxCmd.Stdout = os.Stdout
	tmuxCmd.Stderr = os.Stderr
	if err := tmuxCmd.Run(); err != nil {
		return fmt.Errorf("creating tmux session: %w", err)
	}

	// Send the claude command — unset CLAUDECODE to avoid nested session detection
	time.Sleep(500 * time.Millisecond)
	sendClaude := exec.Command("tmux", "send-keys", "-t", sessionName, "unset CLAUDECODE && "+claudeCmd, "Enter")
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
			// Look for the empty prompt line (claude waits for input)
			lines := strings.Split(strings.TrimSpace(pane), "\n")
			if len(lines) > 0 {
				lastLine := strings.TrimSpace(lines[len(lines)-1])
				// Claude's input prompt is typically an empty line or just ">"
				if lastLine == ">" || lastLine == "❯" || lastLine == "" {
					// Check if claude has finished loading (shows tips or has rendered UI)
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
	// tmux session names can't have dots or colons
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
