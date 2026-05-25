package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: test-project
agent: claude-code
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "test-project" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test-project")
	}
	if cfg.Agent != "claude-code" {
		t.Errorf("Agent = %q, want %q", cfg.Agent, "claude-code")
	}
}

func TestParse_FullConfig(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=abc\n"), 0644)

	yaml := `name: my-app
agent: claude-code
packages:
  - ripgrep
  - jq
runtimes:
  node: "22"
  bun: "latest"
ports:
  - 3000
  - "8080:80"
env:
  NODE_ENV: development
env_file:
  - .env
network:
  - "*.npmjs.org"
  - "api.github.com"
claude:
  plugins:
    marketplaces:
      - anthropics/claude-plugins-official
    install:
      - superpowers@claude-plugins-official
  settings:
    alwaysThinkingEnabled: true
memory: |
  Use bun, not npm.
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Packages) != 2 {
		t.Errorf("Packages len = %d, want 2", len(cfg.Packages))
	}
	if cfg.Runtimes["node"] != "22" {
		t.Errorf("Runtimes[node] = %q, want %q", cfg.Runtimes["node"], "22")
	}
	if len(cfg.Ports) != 2 {
		t.Errorf("Ports len = %d, want 2", len(cfg.Ports))
	}
	if cfg.MergedEnv["NODE_ENV"] != "development" {
		t.Errorf("MergedEnv[NODE_ENV] = %q, want %q", cfg.MergedEnv["NODE_ENV"], "development")
	}
	if cfg.MergedEnv["SECRET"] != "abc" {
		t.Errorf("MergedEnv[SECRET] = %q, want %q", cfg.MergedEnv["SECRET"], "abc")
	}
}

func TestParse_DefaultName(t *testing.T) {
	dir := t.TempDir()
	yaml := `agent: claude-code
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name == "" {
		t.Error("Name should default to directory name, got empty")
	}
}

func TestParse_DefaultAgent(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: my-app
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Agent != "claude-code" {
		t.Errorf("Agent = %q, want default %q", cfg.Agent, "claude-code")
	}
}

func TestValidate_InvalidAgent(t *testing.T) {
	cfg := &Config{Name: "test", Agent: "invalid-agent"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid agent")
	}
}

func TestValidate_TemplateWithPackages(t *testing.T) {
	cfg := &Config{
		Name:     "test",
		Agent:    "claude-code",
		Template: "my-image:v1",
		Packages: []string{"jq"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when template is set with packages")
	}
}

func TestValidate_TemplateWithRuntimes(t *testing.T) {
	cfg := &Config{
		Name:     "test",
		Agent:    "claude-code",
		Template: "my-image:v1",
		Runtimes: map[string]string{"node": "22"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when template is set with runtimes")
	}
}

func TestValidate_InvalidRuntime(t *testing.T) {
	cfg := &Config{
		Name:     "test",
		Agent:    "claude-code",
		Runtimes: map[string]string{"java": "21"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for unsupported runtime")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := &Config{
		Name:  "test",
		Agent: "claude-code",
		Ports: []string{"not-a-port"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid port")
	}
}

func TestDiscover_RootFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "blvckhole.yaml"), []byte("name: test\n"), 0644)

	path, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(path) != "blvckhole.yaml" {
		t.Errorf("got %q, want blvckhole.yaml", filepath.Base(path))
	}
}

func TestDiscover_ConfigDir(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".config", "blvckhole")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "blvckhole.yaml"), []byte("name: test\n"), 0644)

	path, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(configDir, "blvckhole.yaml") {
		t.Errorf("got %q, want config dir path", path)
	}
}

func TestDiscover_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := Discover(dir)
	if err == nil {
		t.Fatal("expected error when no config file exists")
	}
}
