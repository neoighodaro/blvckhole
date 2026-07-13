package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:     "shell",
	Aliases: []string{"ssh"},
	Short:   "Open a shell in the sandbox",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := loadConfig(cwd)
		if err != nil {
			return err
		}

		b, err := loadBackend(cfg)
		if err != nil {
			return err
		}

		if !b.IsRunning(cfg) {
			if err := runStart(b, cfg); err != nil {
				return err
			}
		}

		shellDir := cfg.Shell.Directory
		if shellDir == "" {
			shellDir = cfg.Workspace
		}

		return b.Shell(cfg, shellDir)
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
