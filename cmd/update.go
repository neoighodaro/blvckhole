package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/neoighodaro/blvckhole/internal/update"
	"github.com/spf13/cobra"
)

// updateCheckCmd is the hidden background worker spawned by the root command.
// It refreshes the update-state cache and never prints anything.
var updateCheckCmd = &cobra.Command{
	Use:    "__update-check",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := update.StatePath()
		if err != nil {
			return nil // silent: a background check never surfaces errors
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		_ = update.RunCheck(ctx, update.NewClient(), path, time.Now())
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download and install the latest blvckhole release",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !update.IsRelease(version) {
			return fmt.Errorf("you're on a development build (%s); run 'git pull && make build' to update", version)
		}

		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot determine executable path: %w", err)
		}
		self, err = filepath.EvalSymlinks(self)
		if err != nil {
			return fmt.Errorf("cannot resolve executable path: %w", err)
		}

		client := update.NewClient()
		ctx := cmd.Context()

		rel, err := client.LatestRelease(ctx)
		if err != nil {
			return fmt.Errorf("failed to check latest release: %w", err)
		}
		if !update.IsNewer(version, rel.TagName) {
			fmt.Println(ui.Success.Render("blvckhole is already up to date (" + version + ")."))
			return nil
		}

		fmt.Println(ui.Info.Render("Updating " + version + " -> " + rel.TagName + "..."))

		assetName := update.AssetName(rel.TagName, runtime.GOOS, runtime.GOARCH)
		assetURL, ok := rel.AssetURL(assetName)
		if !ok {
			return fmt.Errorf("no release asset for this platform: %s", assetName)
		}
		sumsURL, ok := rel.AssetURL("checksums.txt")
		if !ok {
			return fmt.Errorf("release is missing checksums.txt")
		}

		tarball, err := client.Download(ctx, assetURL)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", assetName, err)
		}
		checksums, err := client.Download(ctx, sumsURL)
		if err != nil {
			return fmt.Errorf("failed to download checksums: %w", err)
		}
		if err := update.VerifyChecksum(checksums, assetName, tarball); err != nil {
			return fmt.Errorf("integrity check failed: %w", err)
		}

		binary, err := update.ExtractBinary(tarball)
		if err != nil {
			return fmt.Errorf("failed to extract binary: %w", err)
		}
		if err := update.ReplaceBinary(self, binary); err != nil {
			return fmt.Errorf("failed to install update: %w", err)
		}

		fmt.Println(ui.Success.Render("Updated to " + rel.TagName + "."))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCheckCmd)
	rootCmd.AddCommand(updateCmd)
}
