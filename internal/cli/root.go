package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tasm",
	Short: "tasm - tmux agent session manager",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tasm - tmux agent session manager")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(reposCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(cleanupCmd)
}
