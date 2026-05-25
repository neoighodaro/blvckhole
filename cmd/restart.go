package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var restartForce bool

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

		if sandbox.IsRunning(cfg.Name) || sandbox.Exists(cfg.Name) {
			if !restartForce {
				fmt.Println(ui.Warn.Render("Warning: This will destroy the sandbox and all data not part of the project."))
				fmt.Print(ui.Info.Render("Continue? [y/N] "))
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				if strings.TrimSpace(strings.ToLower(answer)) != "y" {
					fmt.Println(ui.Info.Render("Aborted."))
					return nil
				}
			}

			fmt.Println(ui.Accent.Render("Removing sandbox..."))
			if err := sandbox.Remove(cfg.Name); err != nil {
				return fmt.Errorf("failed to remove sandbox: %w", err)
			}
		}

		return runStart(cfg)
	},
}

func init() {
	restartCmd.Flags().BoolVarP(&restartForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(restartCmd)
}
