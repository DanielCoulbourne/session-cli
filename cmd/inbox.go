package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Check reports from spawned sessions",
	Long: `Read reports dropped by spawned Claude Code sessions into ~/orch/inbox/.

Examples:
  session inbox              # List all unread reports
  session inbox --read       # Show full contents of all reports
  session inbox --clear      # Remove all reports after reading`,
	RunE: runInbox,
}

func init() {
	inboxCmd.Flags().Bool("read", false, "Show full contents of all reports")
	inboxCmd.Flags().Bool("clear", false, "Remove all reports after reading")
	inboxCmd.Flags().String("session", "", "Filter reports by session name")
	rootCmd.AddCommand(inboxCmd)
}

func runInbox(cmd *cobra.Command, args []string) error {
	read, _ := cmd.Flags().GetBool("read")
	clear, _ := cmd.Flags().GetBool("clear")
	sessionFilter, _ := cmd.Flags().GetString("session")

	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No inbox directory. No reports yet.")
			return nil
		}
		return err
	}

	var reports []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		reports = append(reports, e.Name())
	}

	sort.Strings(reports)

	if len(reports) == 0 {
		fmt.Println("Inbox empty. No reports from sessions.")
		return nil
	}

	for _, name := range reports {
		path := filepath.Join(inboxDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Filter by session if requested
		if sessionFilter != "" && !strings.Contains(string(content), sessionFilter) && !strings.Contains(name, sessionFilter) {
			continue
		}

		if read {
			fmt.Printf("━━━ %s ━━━\n", name)
			fmt.Println(string(content))
			fmt.Println()
		} else {
			// Show just the first line (title) of each report
			lines := strings.SplitN(string(content), "\n", 3)
			title := name
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && strings.HasPrefix(line, "#") {
					title = strings.TrimPrefix(line, "# ")
					break
				}
			}
			fmt.Printf("  %s  →  %s\n", name, title)
		}
	}

	if clear {
		for _, name := range reports {
			os.Remove(filepath.Join(inboxDir, name))
		}
		fmt.Printf("\nCleared %d report(s).\n", len(reports))
	}

	return nil
}
