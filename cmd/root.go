package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "blvckhole",
	Short:         "Docker Sandbox wrapper with declarative YAML config",
	Long:          ui.RenderLogo() + "\n  A CLI tool to create and manage Docker Sandboxes from a YAML config file.",
	SilenceUsage:  true,
}

func Execute() error {
	return rootCmd.Execute()
}

func ensureSbxInstalled() error {
	if _, err := exec.LookPath("sbx"); err != nil {
		return fmt.Errorf("sbx is not installed. Install it from: https://docs.docker.com/ai/sandboxes/")
	}
	return nil
}

func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, ui.Error.Render("Error: "+msg))
	os.Exit(1)
}
