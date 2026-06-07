package cli

import (
	"fmt"
	"os"

	"github.com/jonschaeffer/tmux-agent-session-manager/internal/deps"
	"github.com/jonschaeffer/tmux-agent-session-manager/internal/session"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <session-name>",
	Short: "Attach to an existing AI session",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := deps.Check(); err != nil {
			return err
		}
		return deps.CheckTmuxRunning()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := session.Attach(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}
