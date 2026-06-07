package cli

import (
	"fmt"
	"sort"

	"github.com/jonschaeffer/tmux-ai-session-manager/internal/config"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List configured agent names",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		names := make([]string, 0, len(cfg.Agents))
		for name := range cfg.Agents {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			fmt.Println(name)
		}
		return nil
	},
}
