package cli

import (
	"fmt"
	"os"

	"github.com/jonschaeffer/tmux-ai-session-manager/internal/deps"
	"github.com/jonschaeffer/tmux-ai-session-manager/internal/session"
	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active AI sessions",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := deps.Check(); err != nil {
			return err
		}
		return deps.CheckTmuxRunning()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessions, err := session.List()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if len(sessions) == 0 {
			return nil
		}

		if listJSON {
			data, err := session.MarshalJSON(sessions)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		for _, s := range sessions {
			fmt.Printf("%-40s %-10s %s\n", s.Name, s.Agent, session.RelativeTime(s.CreatedAt))
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
}
