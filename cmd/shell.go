package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:     "shell",
	Aliases: []string{"ssh"},
	Short:   "Open a shell in the sandbox",
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

		shellDir := cfg.Shell.Directory
		if shellDir == "" {
			shellDir = cfg.Workspace
		}

		shellArgs := []string{"bash"}
		if shellDir != "" {
			quoted := "'" + strings.ReplaceAll(shellDir, "'", "'\\''") + "'"
			shellArgs = []string{"bash", "-c", fmt.Sprintf("cd %s && exec bash", quoted)}
		}

		sbxPath, err := exec.LookPath("sbx")
		if err != nil {
			return fmt.Errorf("sbx not found in PATH: %w", err)
		}

		execArgs := []string{"sbx", "exec", "-it", cfg.Name, "--"}
		execArgs = append(execArgs, shellArgs...)

		return syscall.Exec(sbxPath, execArgs, os.Environ())
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
