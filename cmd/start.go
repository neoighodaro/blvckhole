package cmd

import (
	"fmt"
	"os"

	"github.com/neoighodaro/blvckhole/internal/config"
	"github.com/neoighodaro/blvckhole/internal/kit"
	"github.com/neoighodaro/blvckhole/internal/sandbox"
	"github.com/neoighodaro/blvckhole/internal/template"
	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Build template, generate kit, and create sandbox",
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

		return runStart(cfg)
	},
}

func loadConfig(projectDir string) (*config.Config, error) {
	path, err := config.Discover(projectDir)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Parse(path, projectDir)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func runStart(cfg *config.Config) error {
	if sandbox.IsRunning(cfg.Name) {
		fmt.Println(ui.Success.Render("Sandbox already running (" + cfg.Name + ")"))
		fmt.Println(ui.Info.Render("  Run 'blvckhole agent' to start the AI agent"))
		fmt.Println(ui.Info.Render("  Run 'blvckhole shell' to open a shell"))
		return nil
	}

	if sandbox.Exists(cfg.Name) {
		fmt.Println(ui.Info.Render("Stale sandbox detected, removing..."))
		sandbox.Remove(cfg.Name)
	}

	kitDir := cfg.KitDir()
	if err := os.MkdirAll(kitDir, 0755); err != nil {
		return fmt.Errorf("failed to create kit directory: %w", err)
	}

	if cfg.Template == "" {
		fmt.Println(ui.Accent.Render("Building sandbox image..."))
		if err := template.Build(cfg); err != nil {
			return err
		}

		fmt.Println(ui.Accent.Render("Loading image into sandbox runtime..."))
		if err := template.LoadTemplate(cfg); err != nil {
			return err
		}
	}

	fmt.Println(ui.Accent.Render("Generating kit..."))
	if err := kit.Generate(cfg, kitDir); err != nil {
		return err
	}

	templateImage := cfg.SandboxImageName()
	if cfg.Template != "" {
		templateImage = cfg.Template
	}

	fmt.Println(ui.Accent.Render("Creating sandbox..."))
	if err := sandbox.Create(cfg.Name, templateImage, kitDir, cfg.SbxAgent(), "."); err != nil {
		return fmt.Errorf("failed to create sandbox: %w", err)
	}

	if len(cfg.Network) > 0 {
		fmt.Println(ui.Accent.Render("Applying network whitelist..."))
		if err := sandbox.AllowNetwork(cfg.Name, cfg.Network); err != nil {
			return fmt.Errorf("failed to set network policy: %w", err)
		}
	}

	for _, port := range cfg.Ports {
		fmt.Println(ui.Accent.Render("Publishing port " + port + "..."))
		if err := sandbox.PublishPort(cfg.Name, port); err != nil {
			return fmt.Errorf("failed to publish port %s: %w", port, err)
		}
	}

	fmt.Println(ui.Success.Render("Sandbox started (" + cfg.Name + ")"))
	fmt.Println(ui.Info.Render("  Run 'blvckhole agent' to start the AI agent"))
	fmt.Println(ui.Info.Render("  Run 'blvckhole shell' to open a shell"))
	return nil
}

func init() {
	rootCmd.AddCommand(startCmd)
}
