package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var symlinkFlag bool
var installDirFlag string

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install blvckhole to ~/.local/bin/",
	RunE: func(cmd *cobra.Command, args []string) error {
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot determine executable path: %w", err)
		}
		self, err = filepath.EvalSymlinks(self)
		if err != nil {
			return fmt.Errorf("cannot resolve executable path: %w", err)
		}

		var binDir string
		if installDirFlag != "" {
			binDir = installDirFlag
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot determine home directory: %w", err)
			}
			binDir = filepath.Join(home, ".local", "bin")
		}
		dest := filepath.Join(binDir, "blvckhole")

		if err := os.MkdirAll(binDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", binDir, err)
		}

		os.Remove(dest)

		if symlinkFlag {
			if err := os.Symlink(self, dest); err != nil {
				return fmt.Errorf("failed to symlink: %w", err)
			}
			fmt.Println(ui.Success.Render("Symlinked blvckhole to " + dest + " -> " + self))
			return nil
		}

		src, err := os.Open(self)
		if err != nil {
			return fmt.Errorf("cannot read binary: %w", err)
		}
		defer src.Close()

		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("cannot write to %s: %w", dest, err)
		}
		defer out.Close()

		if _, err := io.Copy(out, src); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}

		fmt.Println(ui.Success.Render("Installed blvckhole to " + dest))
		return nil
	},
}

func init() {
	installCmd.Flags().BoolVarP(&symlinkFlag, "symlink", "s", false, "Symlink to the current binary instead of copying")
	installCmd.Flags().StringVarP(&installDirFlag, "install-directory", "d", "", "Install directory (default: ~/.local/bin/)")
	rootCmd.AddCommand(installCmd)
}
