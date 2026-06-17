package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neoighodaro/blvckhole/internal/config"
)

func TestGenerate_CreatesSpecYaml(t *testing.T) {
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	cfg := &config.Config{
		Name:       "test-app",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{"NODE_ENV": "development"},
		Network:    []string{"*.npmjs.org", "api.github.com"},
		Memory:     "Use bun, not npm.\n",
		ProjectDir: dir,
	}

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	specPath := filepath.Join(kitDir, "spec.yaml")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("spec.yaml not created: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `schemaVersion: "1"`) {
		t.Error("missing schemaVersion")
	}
	if !strings.Contains(content, "kind: mixin") {
		t.Error("missing kind: mixin")
	}
	if !strings.Contains(content, "blvckhole-test-app") {
		t.Error("missing generated kit name")
	}
	if !strings.Contains(content, "NODE_ENV") {
		t.Error("missing environment variable")
	}
	if !strings.Contains(content, "*.npmjs.org") {
		t.Error("missing network domain")
	}
	if !strings.Contains(content, "Use bun") {
		t.Error("missing memory content")
	}
}

func TestGenerate_CreatesThemeFile(t *testing.T) {
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	cfg := &config.Config{
		Name:       "test-app",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{},
		ProjectDir: dir,
	}

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	themePath := filepath.Join(kitDir, "files", "home", ".claude", "themes", "sandbox.json")
	if _, err := os.Stat(themePath); err != nil {
		t.Fatalf("theme file not created: %v", err)
	}

	data, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("cannot read theme file: %v", err)
	}
	if !strings.Contains(string(data), "Catppuccin") {
		t.Error("theme file should contain Catppuccin Mocha theme")
	}
}

func TestGenerate_CreatesSettingsFile(t *testing.T) {
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	cfg := &config.Config{
		Name:       "test-app",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{},
		ProjectDir: dir,
		Claude: config.ClaudeConfig{
			Settings: map[string]interface{}{
				"alwaysThinkingEnabled": true,
			},
		},
	}

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	settingsPath := filepath.Join(kitDir, "files", "home", ".claude", "settings.sandbox.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
	if !strings.Contains(string(data), "alwaysThinkingEnabled") {
		t.Error("settings file should contain merged settings")
	}
}

func TestGenerate_CreatesDashboardFile(t *testing.T) {
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	cfg := &config.Config{
		Name:       "test-app",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{},
		ProjectDir: dir,
	}

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dashPath := filepath.Join(kitDir, "files", "home", ".claude", "claude-dashboard.local.json")
	if _, err := os.Stat(dashPath); err != nil {
		t.Fatalf("dashboard config not created: %v", err)
	}
}

func TestGenerate_CustomTheme(t *testing.T) {
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	customTheme := `{"name": "My Custom Theme", "base": "dark"}`
	themePath := filepath.Join(dir, "my-theme.json")
	os.WriteFile(themePath, []byte(customTheme), 0644)

	cfg := &config.Config{
		Name:       "test-app",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{},
		ProjectDir: dir,
		Claude: config.ClaudeConfig{
			Theme: "./my-theme.json",
		},
	}

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outTheme := filepath.Join(kitDir, "files", "home", ".claude", "themes", "sandbox.json")
	data, err := os.ReadFile(outTheme)
	if err != nil {
		t.Fatalf("theme file not created: %v", err)
	}
	if !strings.Contains(string(data), "My Custom Theme") {
		t.Error("should use custom theme, not default")
	}
}

func TestGenerate_NoEnvNoNetworkNoMemory(t *testing.T) {
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	cfg := &config.Config{
		Name:       "bare",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{},
		ProjectDir: dir,
	}

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(kitDir, "spec.yaml"))
	content := string(data)
	if strings.Contains(content, "environment:") {
		t.Error("should not include environment section when no env vars")
	}
	if strings.Contains(content, "network:") {
		t.Error("should not include network section when no domains")
	}
	if strings.Contains(content, "memory:") {
		t.Error("should not include memory section when empty")
	}
}

func TestGenerate_WritesHandoffSkillWhenEnabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate from any real ~/.claude/skills
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	cfg := &config.Config{
		Name:       "demo",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{},
		ProjectDir: dir,
	}
	cfg.Handoff.Enabled = true

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(kitDir, "files", "home", ".claude", "skills", "sandbox-handoff", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("sandbox-handoff skill not written: %v", err)
	}
	if !strings.Contains(string(data), "name: sandbox-handoff") {
		t.Error("skill file should contain the sandbox-handoff frontmatter")
	}
}

func TestGenerate_SkipsHandoffSkillWhenDisabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate from any real ~/.claude/skills
	dir := t.TempDir()
	kitDir := filepath.Join(dir, ".config", "blvckhole", ".kit")

	cfg := &config.Config{
		Name:       "demo",
		Agent:      "claude-code",
		MergedEnv:  map[string]string{},
		ProjectDir: dir,
	}

	if err := Generate(cfg, kitDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(kitDir, "files", "home", ".claude", "skills", "sandbox-handoff", "SKILL.md")
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Errorf("skill should not be written when handoff disabled (err=%v)", err)
	}
}
