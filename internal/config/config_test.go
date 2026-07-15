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

func TestParse_ScriptsOnCreateAndOnStart(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: scripted
scripts:
  on_create:
    - "composer install"
  on_start:
    - "bash bridge.sh"
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Scripts.OnCreate) != 1 || cfg.Scripts.OnCreate[0] != "composer install" {
		t.Errorf("Scripts.OnCreate = %v, want [composer install]", cfg.Scripts.OnCreate)
	}
	if len(cfg.Scripts.OnStart) != 1 || cfg.Scripts.OnStart[0] != "bash bridge.sh" {
		t.Errorf("Scripts.OnStart = %v, want [bash bridge.sh]", cfg.Scripts.OnStart)
	}
	if cfg.UsedDeprecatedStartup {
		t.Error("UsedDeprecatedStartup should be false when only scripts: is used")
	}
}

func TestParse_DeprecatedStartupFoldsIntoOnCreate(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: legacy
startup:
  - "echo legacy"
scripts:
  on_create:
    - "echo new"
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.UsedDeprecatedStartup {
		t.Error("UsedDeprecatedStartup should be true when startup: is set")
	}
	// Deprecated startup runs before any explicit on_create.
	want := []string{"echo legacy", "echo new"}
	if len(cfg.Scripts.OnCreate) != len(want) {
		t.Fatalf("Scripts.OnCreate = %v, want %v", cfg.Scripts.OnCreate, want)
	}
	for i := range want {
		if cfg.Scripts.OnCreate[i] != want[i] {
			t.Errorf("Scripts.OnCreate[%d] = %q, want %q", i, cfg.Scripts.OnCreate[i], want[i])
		}
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

func TestParse_HandoffDefaultsURL(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: my-app
handoff:
  enabled: true
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Handoff.Enabled {
		t.Fatal("Handoff.Enabled should be true")
	}
	if cfg.Handoff.URL != "http://host.docker.internal:8787" {
		t.Errorf("Handoff.URL = %q, want default", cfg.Handoff.URL)
	}
	if cfg.HandoffPort() != "8787" {
		t.Errorf("HandoffPort() = %q, want 8787", cfg.HandoffPort())
	}
}

func TestParse_HandoffCustomURL(t *testing.T) {
	dir := t.TempDir()
	yaml := `name: my-app
handoff:
  enabled: true
  url: http://host.docker.internal:9000
`
	path := filepath.Join(dir, "blvckhole.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HandoffPort() != "9000" {
		t.Errorf("HandoffPort() = %q, want 9000", cfg.HandoffPort())
	}
}

func TestValidate_HandoffInvalidURL(t *testing.T) {
	cfg := &Config{
		Name:  "test",
		Agent: "claude-code",
		Handoff: HandoffConfig{
			Enabled: true,
			URL:     "not-a-url-without-port",
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for handoff.url without host:port")
	}
}

func TestValidate_HandoffDisabledSkipsURLCheck(t *testing.T) {
	cfg := &Config{
		Name:    "test",
		Agent:   "claude-code",
		Handoff: HandoffConfig{Enabled: false, URL: "garbage"},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("disabled handoff should not validate URL, got: %v", err)
	}
}

func TestParse_HandoffDefaultsURLNonoBackend(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `name: myapp
backend: nono
handoff:
  enabled: true
`)
	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Handoff.URL != "http://localhost:8787" {
		t.Errorf("Handoff.URL = %q, want http://localhost:8787", cfg.Handoff.URL)
	}
	if cfg.HandoffPort() != "8787" {
		t.Errorf("HandoffPort() = %q, want 8787", cfg.HandoffPort())
	}
}

func TestParse_Bridges(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `name: myapp
packages:
  - socat
bridges:
  - name: pgsql
    port: 5432
    host_port: 53432
    env: DB_HOST
  - name: redis
    port: 6379
    host_port: 56379
`)
	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Bridges) != 2 {
		t.Fatalf("Bridges len = %d, want 2", len(cfg.Bridges))
	}
	pg := cfg.Bridges[0]
	if pg.Name != "pgsql" || pg.Port != 5432 || pg.HostPort != 53432 || pg.Env != "DB_HOST" {
		t.Errorf("Bridges[0] = %+v, want {pgsql 5432 53432 DB_HOST}", pg)
	}
	// env is optional.
	if cfg.Bridges[1].Env != "" {
		t.Errorf("Bridges[1].Env = %q, want empty (optional)", cfg.Bridges[1].Env)
	}
}

func TestValidate_BridgeMissingName(t *testing.T) {
	cfg := &Config{Name: "x", Agent: "claude-code", Bridges: []BridgeConfig{{Port: 5432, HostPort: 53432}}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for bridge with no name")
	}
}

func TestValidate_BridgeInvalidPort(t *testing.T) {
	cfg := &Config{Name: "x", Agent: "claude-code", Bridges: []BridgeConfig{{Name: "pgsql", Port: 0, HostPort: 53432}}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for bridge port 0")
	}
	cfg = &Config{Name: "x", Agent: "claude-code", Bridges: []BridgeConfig{{Name: "pgsql", Port: 5432, HostPort: 70000}}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for bridge host_port > 65535")
	}
}

func TestValidate_BridgeInvalidName(t *testing.T) {
	cfg := &Config{Name: "x", Agent: "claude-code", Bridges: []BridgeConfig{{Name: "not a host", Port: 5432, HostPort: 53432}}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for bridge name with spaces")
	}
}

func TestValidate_BridgeWithTemplateErrors(t *testing.T) {
	cfg := &Config{
		Name:     "x",
		Agent:    "claude-code",
		Template: "my/custom:image",
		Bridges:  []BridgeConfig{{Name: "pgsql", Port: 5432, HostPort: 53432}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error: bridges require blvckhole to inject socat, impossible with a custom template")
	}
}

func TestValidate_BridgeValid(t *testing.T) {
	cfg := &Config{
		Name:    "x",
		Agent:   "claude-code",
		Bridges: []BridgeConfig{{Name: "pgsql", Port: 5432, HostPort: 53432, Env: "DB_HOST"}},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid bridge should pass, got: %v", err)
	}
}
