package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

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

	if cfg.UsedDeprecatedStartup {
		fmt.Println(ui.Warn.Render("'startup:' is deprecated — use 'scripts.on_create' (runs once) or 'scripts.on_start' (runs on every start)."))
	}

	return cfg, nil
}

func runStart(cfg *config.Config) error {
	if sandbox.IsRunning(cfg.Name) {
		if configChanged(cfg) {
			fmt.Println(ui.Warn.Render("Config has changed since this sandbox was created."))
			fmt.Print(ui.Info.Render("Restart now? [y/N] "))
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(answer)) == "y" {
				fmt.Println(ui.Accent.Render("Removing sandbox..."))
				if err := sandbox.Remove(cfg.Name); err != nil {
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

	if cfg.Handoff.Enabled {
		cfg.MergedEnv["BLVCKHOLE_SANDBOX"] = cfg.Name
		cfg.MergedEnv["BLVCKHOLE_HANDOFF_URL"] = cfg.Handoff.URL
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

	storeConfigHash(cfg)

	if cfg.Workspace != "" {
		fmt.Println(ui.Accent.Render("Linking project to " + cfg.Workspace + "..."))
		if err := sandbox.LinkWorkspace(cfg.Name, cfg.ProjectDir, cfg.Workspace); err != nil {
			return fmt.Errorf("failed to link project to workspace: %w", err)
		}
	}

	if len(cfg.Network) > 0 {
		fmt.Println(ui.Accent.Render("Applying network whitelist..."))
		if err := sandbox.AllowNetwork(cfg.Name, cfg.Network); err != nil {
			return fmt.Errorf("failed to set network policy: %w", err)
		}
	}

	if cfg.Handoff.Enabled {
		resource := "localhost:" + cfg.HandoffPort()
		fmt.Println(ui.Accent.Render("Allowing handoff broker (" + resource + ")..."))
		if err := sandbox.AllowNetwork(cfg.Name, []string{resource}); err != nil {
			return fmt.Errorf("failed to allow handoff broker network: %w", err)
		}
	}

	for _, port := range cfg.Ports {
		fmt.Println(ui.Accent.Render("Publishing port " + port + "..."))
		if err := sandbox.PublishPort(cfg.Name, port); err != nil {
			return fmt.Errorf("failed to publish port %s: %w", port, err)
		}
	}

	// on_start commands run on every shell/agent session via the per-session
	// init hook (/etc/sandbox-persistent.sh, baked into the image by the
	// Dockerfile), so they are not run here — only on_create runs at creation.
	if len(cfg.Scripts.OnCreate) > 0 {
		workDir := cfg.ProjectDir
		if cfg.Workspace != "" {
			workDir = cfg.Workspace
		}
		for _, cmd := range cfg.Scripts.OnCreate {
			fmt.Println(ui.Accent.Render("Running: " + cmd))
			script := fmt.Sprintf("cd %s && %s", workDir, cmd)
			if err := sandbox.Exec(cfg.Name, false, "bash", "-c", script); err != nil {
				return fmt.Errorf("on_create command failed (%s): %w", cmd, err)
			}
		}
	}

	fmt.Println(ui.Success.Render("Sandbox started (" + cfg.Name + ")"))
	fmt.Println(ui.Info.Render("  Run 'blvckhole agent' to start the AI agent"))
	fmt.Println(ui.Info.Render("  Run 'blvckhole shell' to open a shell"))
	return nil
}

const configHashPath = "/home/agent/.blvckhole-config-hash"

func hashConfigFile(cfg *config.Config) string {
	data, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func storeConfigHash(cfg *config.Config) {
	if hash := hashConfigFile(cfg); hash != "" {
		sandbox.WriteFile(cfg.Name, configHashPath, hash)
	}
}

func configChanged(cfg *config.Config) bool {
	stored, err := sandbox.ReadFile(cfg.Name, configHashPath)
	if err != nil || stored == "" {
		return false
	}
	current := hashConfigFile(cfg)
	if current == "" {
		return false
	}
	return stored != current
}

func init() {
	rootCmd.AddCommand(startCmd)
}
