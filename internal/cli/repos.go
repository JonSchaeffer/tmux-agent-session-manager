package cli

import (
	"fmt"
	"os"

	"github.com/jonschaeffer/tmux-ai-session-manager/internal/config"
	"github.com/jonschaeffer/tmux-ai-session-manager/internal/repo"
	"github.com/spf13/cobra"
)

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List discovered git repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		repos, err := repo.Discover(cfg.RepoRoot)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("repo_root does not exist: %s", cfg.RepoRoot)
			}
			return fmt.Errorf("discovery failed: %w", err)
		}

		repos = repo.Disambiguate(repos)
		for _, r := range repos {
			fmt.Printf("%s\t%s\n", r.Name, r.Path)
		}
		return nil
	},
}
