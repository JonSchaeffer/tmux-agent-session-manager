package cli

import (
	"fmt"

	"github.com/jonschaeffer/tmux-agent-session-manager/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		out, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}

		fmt.Print(string(out))
		return nil
	},
}
