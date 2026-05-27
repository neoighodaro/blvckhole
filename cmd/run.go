package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Execute a command in the sandbox",
	Args:  cobra.MinimumNArgs(1),
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

		workDir := cfg.Workspace
		if workDir == "" {
			workDir = cfg.ProjectDir
		}

		script := fmt.Sprintf("cd '%s' && %s",
			strings.ReplaceAll(workDir, "'", "'\\''"),
			strings.Join(args, " "))

		return sandbox.Exec(cfg.Name, false, "bash", "-c", script)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
