package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("blvckhole version %s\n", version)
		return nil
	},
}

func init() {
	// Register a "version" flag with a -V shorthand. Cobra detects the
	// pre-existing flag and uses it for its built-in --version handling,
	// so both --version and -V print rootCmd.Version automatically.
	rootCmd.Flags().BoolP("version", "V", false, "version for blvckhole")
	rootCmd.AddCommand(versionCmd)
}
