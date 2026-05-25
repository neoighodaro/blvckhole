package cmd

import (
	"errors"
	"os"
	"os/exec"

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

		if err := sandbox.Exec(cfg.Name, true, "bash"); err != nil {
			var exitErr *exec.ExitError
			if ok := errors.As(err, &exitErr); ok {
				os.Exit(exitErr.ExitCode())
			}
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
