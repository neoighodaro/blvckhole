package cmd

import (
	"fmt"
	"os"

	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Stop and start the sandbox",
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

		if sandbox.IsRunning(cfg.Name) {
			fmt.Println(ui.Accent.Render("Stopping sandbox..."))
			if err := sandbox.Remove(cfg.Name); err != nil {
				return fmt.Errorf("failed to remove sandbox: %w", err)
			}
		}

		return runStart(cfg)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
