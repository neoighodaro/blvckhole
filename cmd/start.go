package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/backend"
	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Build template, generate kit, and create sandbox",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg, err := loadConfig(cwd)
		if err != nil {
			return err
		}

		b, err := loadBackend(cfg)
		if err != nil {
			return err
		}

		return runStart(b, cfg)
	},
}

func loadConfig(projectDir string) (*config.Config, error) {
	path, err := config.Resolve(projectDir, configPath)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Parse(path, projectDir)
	if err != nil {
		return nil, err
	}

	if cfg.UsedDeprecatedStartup {
		fmt.Println(ui.Warn.Render("'startup:' is deprecated — use 'scripts.on_create' (runs once) or 'scripts.on_start' (runs on every start)."))
	}

	return cfg, nil
}

func runStart(b backend.Backend, cfg *config.Config) error {
	if b.IsRunning(cfg) {
		if configChanged(b, cfg) {
			fmt.Println(ui.Warn.Render("Config has changed since this sandbox was created."))
			fmt.Print(ui.Info.Render("Restart now? [y/N] "))
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(answer)) == "y" {
				fmt.Println(ui.Accent.Render("Removing sandbox..."))
				if err := b.Remove(cfg); err != nil {
					return fmt.Errorf("failed to remove sandbox: %w", err)
				}
			} else {
				return nil
			}
		} else {
			fmt.Println(ui.Success.Render("Sandbox already running (" + cfg.Name + ")"))
			fmt.Println(ui.Info.Render("  Run 'blvckhole agent' to start the AI agent"))
			fmt.Println(ui.Info.Render("  Run 'blvckhole shell' to open a shell"))
			return nil
		}
	}

	if cfg.Handoff.Enabled {
		cfg.MergedEnv["BLVCKHOLE_SANDBOX"] = cfg.Name
		cfg.MergedEnv["BLVCKHOLE_HANDOFF_URL"] = cfg.Handoff.URL
	}

	if err := b.Provision(cfg); err != nil {
		return err
	}

	fmt.Println(ui.Success.Render("Sandbox started (" + cfg.Name + ")"))
	fmt.Println(ui.Info.Render("  Run 'blvckhole agent' to start the AI agent"))
	fmt.Println(ui.Info.Render("  Run 'blvckhole shell' to open a shell"))
	return nil
}

func configChanged(b backend.Backend, cfg *config.Config) bool {
	stored, err := b.ReadConfigHash(cfg)
	if err != nil || stored == "" {
		return false
	}
	current := cfg.FileHash()
	if current == "" {
		return false
	}
	return stored != current
}

func init() {
	rootCmd.AddCommand(startCmd)
}
