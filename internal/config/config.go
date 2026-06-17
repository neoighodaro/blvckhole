package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/neoighodaro/blvckhole/internal/envfile"
	"gopkg.in/yaml.v3"
)

var validAgents = map[string]bool{
	"claude-code":  true,
	"codex":        true,
	"copilot":      true,
	"cursor":       true,
	"docker-agent": true,
	"droid":        true,
	"gemini":       true,
	"kiro":         true,
	"opencode":     true,
	"shell":        true,
}

var validRuntimes = map[string]bool{
	"node":   true,
	"pnpm":   true,
	"bun":    true,
	"python": true,
	"go":     true,
	"php":    true,
	"rust":   true,
}

var nameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type ClaudePlugins struct {
	Marketplaces []string `yaml:"marketplaces"`
	Install      []string `yaml:"install"`
}

type ShellConfig struct {
	Directory string            `yaml:"directory"`
	Aliases   map[string]string `yaml:"aliases"`
}

type ClaudeConfig struct {
	Theme    string                 `yaml:"theme"`
	Plugins  ClaudePlugins          `yaml:"plugins"`
	Settings map[string]interface{} `yaml:"settings"`
}

type PhpConfig struct {
	Extensions []string `yaml:"extensions"`
}

type ZellijConfig struct {
	DisplayName string `yaml:"display_name"`
}

type HandoffConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

// ScriptsConfig holds commands run inside the sandbox at lifecycle points.
//
//	OnCreate runs once, right after the sandbox is created.
//	OnStart  runs on every container start, including resume after a stop —
//	         use it for things that don't survive a restart (port bridges,
//	         background daemons). Emitted as sbx kit startup commands.
type ScriptsConfig struct {
	OnCreate []string `yaml:"on_create"`
	OnStart  []string `yaml:"on_start"`
}

type Config struct {
	Name      string            `yaml:"name"`
	Agent     string            `yaml:"agent"`
	Template  string            `yaml:"template"`
	Workspace string            `yaml:"workspace"`
	Packages  []string          `yaml:"packages"`
	Runtimes  map[string]string `yaml:"runtimes"`
	Ports     []string          `yaml:"ports"`
	Env       map[string]string `yaml:"env"`
	EnvFile   []string          `yaml:"env_file"`
	Network   []string          `yaml:"network"`
	Scripts   ScriptsConfig     `yaml:"scripts"`
	Startup   []string          `yaml:"startup"` // deprecated: use scripts.on_create
	Shell     ShellConfig       `yaml:"shell"`
	Claude    ClaudeConfig      `yaml:"claude"`
	Php       PhpConfig         `yaml:"php"`
	Zellij    ZellijConfig      `yaml:"zellij"`
	Memory    string            `yaml:"memory"`
	Handoff   HandoffConfig     `yaml:"handoff"`

	MergedEnv             map[string]string `yaml:"-"`
	ProjectDir            string            `yaml:"-"`
	ConfigPath            string            `yaml:"-"`
	UsedDeprecatedStartup bool              `yaml:"-"`
}

// Discover searches for a blvckhole.yaml config file in projectDir.
// It checks the root first, then .config/blvckhole/.
func Discover(projectDir string) (string, error) {
	rootPath := filepath.Join(projectDir, "blvckhole.yaml")
	if _, err := os.Stat(rootPath); err == nil {
		return rootPath, nil
	}

	configPath := filepath.Join(projectDir, ".config", "blvckhole", "blvckhole.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	return "", fmt.Errorf("no blvckhole.yaml found. Run 'blvckhole init' to create one")
}

// Parse reads and unmarshals the YAML config at path, applies defaults,
// merges env_file entries, and validates the result.
func Parse(path string, projectDir string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	cfg.ProjectDir = projectDir
	cfg.ConfigPath = path

	if cfg.Name == "" {
		cfg.Name = filepath.Base(projectDir)
	}
	if cfg.Agent == "" {
		cfg.Agent = "claude-code"
	}

	if cfg.Handoff.Enabled && cfg.Handoff.URL == "" {
		cfg.Handoff.URL = "http://host.docker.internal:8787"
	}

	// Deprecated 'startup:' is an alias for 'scripts.on_create' — fold it in
	// (running before any explicit on_create) and flag it so callers can warn.
	if len(cfg.Startup) > 0 {
		cfg.UsedDeprecatedStartup = true
		cfg.Scripts.OnCreate = append(append([]string{}, cfg.Startup...), cfg.Scripts.OnCreate...)
	}

	cfg.MergedEnv = make(map[string]string)
	for _, envPath := range cfg.EnvFile {
		absPath := envPath
		if !filepath.IsAbs(envPath) {
			absPath = filepath.Join(projectDir, envPath)
		}
		parsed, err := envfile.Parse(absPath)
		if err != nil {
			return nil, err
		}
		for k, v := range parsed {
			cfg.MergedEnv[k] = v
		}
	}
	for k, v := range cfg.Env {
		cfg.MergedEnv[k] = v
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks the config for semantic errors.
func (c *Config) Validate() error {
	if c.Name != "" && !nameRegexp.MatchString(c.Name) {
		return fmt.Errorf("invalid name %q: must be lowercase alphanumeric with hyphens", c.Name)
	}

	if !validAgents[c.Agent] {
		agents := make([]string, 0, len(validAgents))
		for a := range validAgents {
			agents = append(agents, a)
		}
		return fmt.Errorf("invalid agent %q: must be one of: %s", c.Agent, strings.Join(agents, ", "))
	}

	if c.Template != "" && len(c.Packages) > 0 {
		return fmt.Errorf("cannot set both 'template' and 'packages': when using a custom template, packages must be installed in your Dockerfile")
	}
	if c.Template != "" && len(c.Runtimes) > 0 {
		return fmt.Errorf("cannot set both 'template' and 'runtimes': when using a custom template, runtimes must be installed in your Dockerfile")
	}

	for name := range c.Runtimes {
		if !validRuntimes[name] {
			runtimes := make([]string, 0, len(validRuntimes))
			for r := range validRuntimes {
				runtimes = append(runtimes, r)
			}
			return fmt.Errorf("unsupported runtime %q: must be one of: %s", name, strings.Join(runtimes, ", "))
		}
	}

	if _, hasPnpm := c.Runtimes["pnpm"]; hasPnpm {
		if _, hasNode := c.Runtimes["node"]; !hasNode {
			return fmt.Errorf("pnpm requires the node runtime: add 'node' to your runtimes")
		}
	}

	for _, port := range c.Ports {
		if err := validatePort(port); err != nil {
			return err
		}
	}

	if c.Handoff.Enabled {
		u, err := url.Parse(c.Handoff.URL)
		if err != nil || u.Host == "" || u.Port() == "" {
			return fmt.Errorf("invalid handoff.url %q: must be a URL with host and port", c.Handoff.URL)
		}
	}

	return nil
}

func validatePort(port string) error {
	parts := strings.SplitN(port, ":", 2)
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return fmt.Errorf("invalid port %q: must be a number between 1 and 65535", port)
		}
	}
	return nil
}

// SbxAgent maps the configured agent name to the sbx CLI agent identifier.
func (c *Config) SbxAgent() string {
	if c.Agent == "claude-code" {
		return "claude"
	}
	return c.Agent
}

// SandboxImageName returns the Docker image name for the sandbox.
func (c *Config) SandboxImageName() string {
	return c.Name + "-sandbox"
}

// KitDir returns the path to the kit directory inside the project config.
func (c *Config) KitDir() string {
	return filepath.Join(c.ProjectDir, ".config", "blvckhole", ".kit")
}

// HandoffPort returns the port from the configured handoff URL, or "" if it
// cannot be parsed.
func (c *Config) HandoffPort() string {
	u, err := url.Parse(c.Handoff.URL)
	if err != nil {
		return ""
	}
	return u.Port()
}
