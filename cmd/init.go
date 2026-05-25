package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/ui"
	"github.com/spf13/cobra"
)

var starterConfig = `# blvckhole sandbox configuration
# Docs: https://github.com/neoighodaro/blvckhole

name: %s
agent: claude-code

# Custom template image (skips Dockerfile generation when set).
# When set, 'packages' and 'runtimes' are ignored — install them in your Dockerfile.
# template: docker.io/my-org/custom-template:v1

# System packages installed via apt-get
packages: []
  # - ripgrep
  # - jq
  # - xz-utils

# Language runtimes
runtimes: {}
  # node: "24"
  # pnpm: "11"
  # bun: "latest"
  # python: "3.12"
  # go: "1.23"
  # php: "8.4"
  # rust: "stable"

# Ports exposed from sandbox to host
ports: []
  # - 3000
  # - "8080:80"

# Inline environment variables
env: {}
  # NODE_ENV: development

# Environment files (loaded in order, later files override earlier ones)
env_file: []
  # - .env
  # - .env.sandbox

# Shell configuration (ZSH is the default shell)
shell:
  # Default working directory when opening a shell
  # directory: /home/agent/project/src

  # Custom aliases (added alongside built-in aliases)
  aliases: {}
    # g: "git"
    # dc: "docker compose"
    # clr: "clear"

# Network whitelist (only these domains are reachable from the sandbox)
network: []
  # - "*.npmjs.org"
  # - "registry.yarnpkg.com"
  # - "api.github.com"

# Claude Code agent customization
claude:
  # Theme file path (Catppuccin Mocha is used by default)
  # theme: ./my-custom-theme.json

  # Plugins to install
  plugins:
    marketplaces: []
      # - anthropics/claude-plugins-official
      # - uppinote20/claude-dashboard
    install: []
      # - superpowers@claude-plugins-official
      # - claude-dashboard@claude-dashboard

  # Settings merged into the agent's settings
  settings: {}
    # alwaysThinkingEnabled: true

# Instructions injected into the agent's memory file (CLAUDE.md)
# memory: |
#   Project-specific instructions here.
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter blvckhole config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}

		configDir := filepath.Join(cwd, ".config", "blvckhole")
		configPath := filepath.Join(configDir, "blvckhole.yaml")
		kitDir := filepath.Join(configDir, ".kit")

		if _, err := os.Stat(configPath); err == nil {
			fmt.Println(ui.Info.Render("Config already exists at " + configPath))
			return nil
		}

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		if err := os.MkdirAll(kitDir, 0755); err != nil {
			return fmt.Errorf("failed to create kit directory: %w", err)
		}

		projectName := filepath.Base(cwd)
		content := fmt.Sprintf(starterConfig, projectName)
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		if err := updateGitignore(cwd); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}

		fmt.Println(ui.Success.Render("Created " + configPath))
		fmt.Println(ui.Info.Render("Edit the config, then run 'blvckhole start' to create your sandbox."))
		return nil
	},
}

func updateGitignore(projectDir string) error {
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	entry := ".config/blvckhole/.kit/"
	marker := "# blvckhole"

	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(existing)
	if strings.Contains(content, entry) {
		return nil
	}

	var addition string
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		addition = "\n"
	}
	addition += "\n" + marker + "\n" + entry + "\n"

	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(addition)
	return err
}

func init() {
	rootCmd.AddCommand(initCmd)
}
