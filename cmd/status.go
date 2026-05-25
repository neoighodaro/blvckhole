package cmd

import (
	"fmt"
	"os"

	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sandbox status",
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

		info, err := sandbox.Status(cfg.Name)
		if err != nil {
			return err
		}

		if info == nil {
			fmt.Println(ui.Info.Render("Sandbox not created yet."))
			fmt.Println(ui.Info.Render("  Run 'blvckhole start' to create it."))
			return nil
		}

		fmt.Println(ui.Bold.Render("Sandbox: ") + cfg.Name)
		fmt.Println(ui.Bold.Render("Status:  ") + info.Status)
		fmt.Println(ui.Bold.Render("Agent:   ") + info.Agent)
		if info.Ports != "" {
			fmt.Println(ui.Bold.Render("Ports:   ") + info.Ports)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
