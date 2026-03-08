package cmd

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active Claude Code sessions",
	Long: `Show all active tmux sessions with heartbeat status.

Examples:
  session list`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

type heartbeatInfo struct {
	Status      string
	LastActive  string
	CurrentTask string
}

func parseHeartbeatFile(path string) (*heartbeatInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := &heartbeatInfo{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status:") {
			info.Status = strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
		} else if strings.HasPrefix(line, "Last active:") {
			info.LastActive = strings.TrimSpace(strings.TrimPrefix(line, "Last active:"))
		} else if strings.HasPrefix(line, "Current task:") {
			info.CurrentTask = strings.TrimSpace(strings.TrimPrefix(line, "Current task:"))
		}
	}
	return info, nil
}

func formatTimeAgo(isoTimestamp string) string {
	// Try parsing common ISO 8601 formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05",
	}

	var t time.Time
	var err error
	for _, f := range formats {
		t, err = time.Parse(f, isoTimestamp)
		if err == nil {
			break
		}
	}
	if err != nil {
		return isoTimestamp
	}

	dur := time.Since(t)
	if dur < 0 {
		return "just now"
	}

	seconds := dur.Seconds()
	if seconds < 60 {
		return fmt.Sprintf("%ds ago", int(seconds))
	}
	minutes := math.Floor(dur.Minutes())
	if minutes < 60 {
		return fmt.Sprintf("%.0fm ago", minutes)
	}
	hours := math.Floor(dur.Hours())
	if hours < 24 {
		return fmt.Sprintf("%.0fh ago", hours)
	}
	days := math.Floor(hours / 24)
	return fmt.Sprintf("%.0fd ago", days)
}

func getHeartbeat(sessionName string) *heartbeatInfo {
	path := filepath.Join(inboxDir, fmt.Sprintf("heartbeat-%s.md", sessionName))
	info, err := parseHeartbeatFile(path)
	if err != nil {
		return nil
	}
	return info
}

func runList(cmd *cobra.Command, args []string) error {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
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

	fmt.Printf("%-20s %-12s %-14s %s\n", "SESSION", "STATUS", "LAST ACTIVE", "CURRENT TASK")
	for _, sessionName := range strings.Split(lines, "\n") {
		sessionName = strings.TrimSpace(sessionName)
		if sessionName == "" {
			continue
		}

		hb := getHeartbeat(sessionName)
		if hb != nil {
			lastActive := formatTimeAgo(hb.LastActive)
			status := hb.Status
			if status == "" {
				status = "unknown"
			}
			task := hb.CurrentTask
			if task == "" {
				task = "-"
			}
			fmt.Printf("%-20s %-12s %-14s %s\n", sessionName, status, lastActive, task)
		} else {
			fmt.Printf("%-20s %-12s %-14s %s\n", sessionName, "-", "(no heartbeat)", "-")
		}
	}
	return nil
}
