package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	homeDir, _ = os.UserHomeDir()
	srcDir     = filepath.Join(homeDir, "src")
	orchDir    = filepath.Join(homeDir, "orch")
	inboxDir   = filepath.Join(orchDir, "inbox")
)

var rootCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage Claude Code sessions for projects",
	Long: `Launch and manage Claude Code sessions in tmux for your projects.

Projects are cloned to ~/src/{reponame} and Claude Code is launched
in a named tmux session. Sessions have access to shared tooling (linear CLI, etc.)
and can report back to the orchestrator via ~/orch/inbox/.

Examples:
  session start kathunk/tidy                    # Clone (if needed) and launch
  session start kathunk/tidy -p "work on THU-1613"  # Launch with a prompt
  session list                                  # Show active sessions
  session attach tidy                           # Attach to a session
  session inbox                                 # Check reports from sessions
  session stop tidy                             # Kill a session`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
