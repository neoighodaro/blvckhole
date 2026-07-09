package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Execute a command in the sandbox",
	Args:  cobra.MinimumNArgs(1),
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

		return b.RunCommand(cfg, strings.Join(args, " "))
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
