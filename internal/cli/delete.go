package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jonschaeffer/tmux-ai-session-manager/internal/deps"
	"github.com/jonschaeffer/tmux-ai-session-manager/internal/session"
	"github.com/spf13/cobra"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete <session-name>",
	Short: "Delete an AI session",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := deps.Check(); err != nil {
			return err
		}
		return deps.CheckTmuxRunning()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionName := args[0]

		if !deleteForce && !promptConfirm(fmt.Sprintf("Delete session %s? This will remove the worktree. [y/N]", sessionName)) {
			return nil
		}

		if err := session.Delete(sessionName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("Deleted session %s\n", sessionName)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "Skip confirmation prompt")
}

func promptConfirm(msg string) bool {
	fmt.Print(msg + " ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(scanner.Text())
		return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
	}
	return false
}
