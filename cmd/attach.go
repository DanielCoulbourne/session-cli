package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach [session-name]",
	Short: "Attach to a Claude Code session",
	Long: `Attach to an existing tmux session.

Examples:
  session attach tidy`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !sessionExists(name) {
		return fmt.Errorf("session %q does not exist. Run 'session list' to see active sessions", name)
	}

	tmuxCmd := exec.Command("tmux", "attach-session", "-t", name)
	tmuxCmd.Stdin = os.Stdin
	tmuxCmd.Stdout = os.Stdout
	tmuxCmd.Stderr = os.Stderr
	return tmuxCmd.Run()
}
