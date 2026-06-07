package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "taism",
	Short: "taism - tmux ai session manager",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("taism - tmux ai session manager")
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
}
