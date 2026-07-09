package cmd

import (
	"fmt"
	"os"

	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the sandbox",
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
			fmt.Println(ui.Info.Render("Sandbox is not running."))
			return nil
		}

		fmt.Println(ui.Accent.Render("Stopping sandbox..."))
		if err := b.Stop(cfg); err != nil {
			return fmt.Errorf("failed to stop sandbox: %w", err)
		}

		fmt.Println(ui.Success.Render("Sandbox stopped (" + cfg.Name + ")"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
