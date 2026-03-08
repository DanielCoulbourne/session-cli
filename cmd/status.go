package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [session-name]",
	Short: "Peek at what a session is currently doing",
	Long: `Capture the current screen contents of a tmux session to see
what Claude Code is working on without attaching.

Examples:
  session status tidy
  session status tidy --lines 50`,
	Args: cobra.ExactArgs(1),
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().Int("lines", 30, "Number of lines to capture")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	name := args[0]
	lines, _ := cmd.Flags().GetInt("lines")

	if !sessionExists(name) {
		return fmt.Errorf("session %q does not exist", name)
	}

	out, err := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-S", fmt.Sprintf("-%d", lines)).Output()
	if err != nil {
		return fmt.Errorf("capturing pane: %w", err)
	}

	content := strings.TrimSpace(string(out))
	if content == "" {
		fmt.Printf("Session %q: (empty screen)\n", name)
		return nil
	}

	fmt.Printf("━━━ %s ━━━\n", name)
	fmt.Println(content)
	fmt.Println("━━━━━━━━━━━━")
	return nil
}
