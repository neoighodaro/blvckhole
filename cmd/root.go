package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/neoighodaro/blvckhole/internal/update"
	"github.com/spf13/cobra"
)

// version is the build version, overridable at build time via
// -ldflags "-X github.com/neoighodaro/blvckhole/cmd.version=v1.2.3".
var version = "dev"

// pendingUpdate holds the newer version discovered by a previous background
// check, if any. Set in PersistentPreRunE, printed after the command runs.
var pendingUpdate string

func stdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// maybeCheckForUpdate reads the cached update state, schedules a background
// refresh if due, and records any available newer version for later notice.
// It must never block on the network or return an error to the caller.
func maybeCheckForUpdate(cmd *cobra.Command) {
	if update.Suppressed(os.Getenv, stdoutIsTTY(), cmd.Name()) {
		return
	}
	path, err := update.StatePath()
	if err != nil {
		return
	}
	state := update.LoadState(path)
	if update.IsNewer(version, state.LatestVersion) {
		pendingUpdate = state.LatestVersion
	}
	if state.Due(time.Now(), update.CheckInterval) {
		if self, err := os.Executable(); err == nil {
			_ = update.Spawn(self)
		}
	}
}

func printUpdateNotice() {
	if pendingUpdate == "" {
		return
	}
	fmt.Fprintln(os.Stderr, ui.Warn.Render(
		"A new version of blvckhole is available: "+version+" -> "+pendingUpdate))
	fmt.Fprintln(os.Stderr, ui.Info.Render("  Run 'blvckhole update' to upgrade."))
}

var rootCmd = &cobra.Command{
	Use:          "blvckhole",
	Short:        "Docker Sandbox wrapper with declarative YAML config",
	Long:         ui.RenderLogo() + "\n  A CLI tool to create and manage Docker Sandboxes from a YAML config file.",
	Version:      version,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		maybeCheckForUpdate(cmd)
		return nil
	},
}

func Execute() error {
	err := rootCmd.Execute()
	printUpdateNotice()
	return err
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
