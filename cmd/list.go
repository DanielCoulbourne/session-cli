package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active Claude Code sessions",
	Long: `Show all active tmux sessions.

Examples:
  session list`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}\t#{session_path}\t#{session_created_string}\t#{session_windows} windows").Output()
	if err != nil {
		// tmux returns error when no server is running
		fmt.Println("No active sessions.")
		return nil
	}

	lines := strings.TrimSpace(string(out))
	if lines == "" {
		fmt.Println("No active sessions.")
		return nil
	}

	fmt.Printf("%-20s %-40s %-25s %s\n", "SESSION", "PATH", "CREATED", "WINDOWS")
	for _, line := range strings.Split(lines, "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) == 4 {
			fmt.Printf("%-20s %-40s %-25s %s\n", parts[0], parts[1], parts[2], parts[3])
		}
	}
	return nil
}
