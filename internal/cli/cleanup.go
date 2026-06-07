package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jonschaeffer/tmux-agent-session-manager/internal/deps"
	"github.com/jonschaeffer/tmux-agent-session-manager/internal/session"
	"github.com/spf13/cobra"
)

var (
	cleanupForce     bool
	cleanupOlderThan string
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Delete all idle AI sessions",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := deps.Check(); err != nil {
			return err
		}
		return deps.CheckTmuxRunning()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessions, err := session.List()
		if err != nil {
			return err
		}

		var minAge time.Duration
		if cleanupOlderThan != "" {
			minAge, err = parseDuration(cleanupOlderThan)
			if err != nil {
				return fmt.Errorf("invalid --older-than value: %w", err)
			}
		}

		var idle []session.Session
		for _, s := range sessions {
			if s.Status != "idle" {
				continue
			}
			if minAge > 0 && time.Since(s.CreatedAt) < minAge {
				continue
			}
			idle = append(idle, s)
		}

		if len(idle) == 0 {
			fmt.Println("No idle sessions found.")
			return nil
		}

		fmt.Printf("Found %d idle session(s):\n", len(idle))
		for _, s := range idle {
			fmt.Printf("  %-40s %-10s %s\n", s.Name, "idle", session.RelativeTime(s.CreatedAt))
		}

		if !cleanupForce {
			fmt.Printf("\nDelete all %d idle session(s)? [y/N] ", len(idle))
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() || !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
				return nil
			}
		}

		deleted, errCount := 0, 0
		for _, s := range idle {
			if err := session.Delete(s.Name); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to delete %s: %s\n", s.Name, err)
				errCount++
			} else {
				fmt.Printf("Deleted %s\n", s.Name)
				deleted++
			}
		}

		fmt.Printf("\nDeleted %d session(s), %d error(s)\n", deleted, errCount)
		return nil
	},
}

func init() {
	cleanupCmd.Flags().BoolVar(&cleanupForce, "force", false, "Skip confirmation prompt")
	cleanupCmd.Flags().StringVar(&cleanupOlderThan, "older-than", "", "Only clean up sessions older than this duration (e.g. 1h, 30m, 7d)")
}

// parseDuration extends time.ParseDuration to support days (e.g. "7d").
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(n, "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid day count in %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
