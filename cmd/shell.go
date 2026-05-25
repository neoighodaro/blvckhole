package cmd

import (
	"os"

	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open a shell in the sandbox",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureSbxInstalled(); err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := loadConfig(cwd)
		if err != nil {
			return err
		}

		if !sandbox.IsRunning(cfg.Name) {
			if err := runStart(cfg); err != nil {
				return err
			}
		}

		return sandbox.Exec(cfg.Name, true, "bash")
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
