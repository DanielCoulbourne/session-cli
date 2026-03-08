package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send [session-name] [message...]",
	Short: "Send a message to a running session",
	Long: `Send text input to a running tmux session. Useful for sending
prompts to a Claude Code session from the orchestrator.

Examples:
  session send tidy "fix the login bug"
  session send tidy "run the test suite"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runSend,
}

func init() {
	rootCmd.AddCommand(sendCmd)
}

func runSend(cmd *cobra.Command, args []string) error {
	name := args[0]
	message := strings.Join(args[1:], " ")

	if !sessionExists(name) {
		return fmt.Errorf("session %q does not exist", name)
	}

	// Send keys to the tmux session
	tmuxCmd := exec.Command("tmux", "send-keys", "-t", name, message, "Enter")
	if err := tmuxCmd.Run(); err != nil {
		return fmt.Errorf("sending to session: %w", err)
	}

	fmt.Printf("Sent to %q: %s\n", name, message)
	return nil
}
