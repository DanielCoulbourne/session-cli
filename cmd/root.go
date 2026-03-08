package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const srcDir = "/Users/clawb/src"

var rootCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage Claude Code sessions for projects",
	Long: `Launch and manage Claude Code sessions in tmux for your projects.

Projects are cloned to ~/src/{reponame} and Claude Code is launched
in a named tmux session.

Examples:
  session start kathunk/tidy                    # Clone (if needed) and launch
  session start kathunk/tidy -p "work on THU-1613"  # Launch with a prompt
  session list                                  # Show active sessions
  session attach tidy                           # Attach to a session
  session stop tidy                             # Kill a session`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
