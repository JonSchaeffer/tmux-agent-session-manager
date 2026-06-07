package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonschaeffer/tmux-ai-session-manager/internal/config"
	"github.com/jonschaeffer/tmux-ai-session-manager/internal/deps"
	"github.com/jonschaeffer/tmux-ai-session-manager/internal/session"
	"github.com/spf13/cobra"
)

var (
	createRepo   string
	createBranch string
	createAgent  string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new AI session",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := deps.Check(); err != nil {
			return err
		}
		return deps.CheckTmuxRunning()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if createRepo == "" || createBranch == "" {
			cmd.Usage()
			fmt.Fprintln(os.Stderr, "\nerror: --repo and --branch are required")
			os.Exit(1)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		agent := createAgent
		if agent == "" {
			agent = cfg.DefaultAgent
		}

		repoName := filepath.Base(createRepo)

		if err := session.Create(session.CreateOpts{
			RepoPath: createRepo,
			RepoName: repoName,
			Branch:   createBranch,
			Agent:    agent,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("Created session ai/%s/%s\n", repoName, createBranch)
		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createRepo, "repo", "", "Path to the git repository")
	createCmd.Flags().StringVar(&createBranch, "branch", "", "Branch/worktree name to create")
	createCmd.Flags().StringVar(&createAgent, "agent", "", "Agent command to launch (default: config default_agent)")
}
