package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [session-name]",
	Short: "Stop a Claude Code session",
	Long: `Kill a tmux session.

Examples:
  session stop tidy
  session stop --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStop,
}

func init() {
	stopCmd.Flags().Bool("all", false, "Stop all sessions")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")

	if all {
		killCmd := exec.Command("tmux", "kill-server")
		if err := killCmd.Run(); err != nil {
			fmt.Println("No active sessions to stop.")
			return nil
		}
		fmt.Println("All sessions stopped.")
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("specify a session name or use --all")
	}

	name := args[0]
	if !sessionExists(name) {
		return fmt.Errorf("session %q does not exist", name)
	}

	killCmd := exec.Command("tmux", "kill-session", "-t", name)
	if err := killCmd.Run(); err != nil {
		return fmt.Errorf("stopping session: %w", err)
	}

	fmt.Printf("Session %q stopped.\n", name)
	return nil
}
