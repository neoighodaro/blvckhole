package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neoighodaro/blvckhole/internal/config"
)

// stubHost replaces the host probes for the duration of a test. look
// defaults to "everything installed"; ver defaults to failing loudly so
// tests that must not check versions catch accidental calls.
func stubHost(t *testing.T, look func(string) (string, error), ver func(string, string) (string, error)) {
	t.Helper()
	origLook, origVer := nonoLookPath, nonoRunVersion
	nonoLookPath, nonoRunVersion = look, ver
	t.Cleanup(func() { nonoLookPath, nonoRunVersion = origLook, origVer })
}

func found(bin string) (string, error) { return "/usr/bin/" + bin, nil }

func noVersionCalls(t *testing.T) func(string, string) (string, error) {
	return func(bin, arg string) (string, error) {
		t.Fatalf("unexpected version probe: %s %s", bin, arg)
		return "", nil
	}
}

func nonoCfg() *config.Config {
	return &config.Config{
		Name:       "myapp",
		Agent:      "claude-code",
		Backend:    "nono",
		ConfigPath: "/proj/blvckhole.yaml",
		ProjectDir: "/proj",
	}
}

func TestValidateForNonoErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr string
	}{
		{"packages", func(c *config.Config) { c.Packages = []string{"jq"} },
			"nono cannot install apt packages on the host; remove 'packages:' or install them yourself"},
		{"template", func(c *config.Config) { c.Template = "my-image" },
			"nono does not use Docker images; remove 'template:'"},
		{"ports", func(c *config.Config) { c.Ports = []string{"3000:3000"} },
			"nono sessions are host processes; ports need no publishing — remove 'ports:'"},
		{"unsupported agent", func(c *config.Config) { c.Agent = "gemini" },
			`agent "gemini" is not supported by the nono backend; supported: claude-code, codex, opencode, shell`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubHost(t, found, noVersionCalls(t))
			cfg := nonoCfg()
			tt.mutate(cfg)
			warnings, err := validateForNono(cfg)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if warnings != nil {
				t.Fatalf("warnings = %v, want nil on error", warnings)
			}
		})
	}
}

func TestValidateForNonoRuntimeMissing(t *testing.T) {
	stubHost(t, func(bin string) (string, error) {
		return "", &missingErr{bin}
	}, noVersionCalls(t))
	cfg := nonoCfg()
	cfg.Runtimes = map[string]string{"go": ""}
	_, err := validateForNono(cfg)
	want := `runtime "go" declared in /proj/blvckhole.yaml but "go" was not found on this machine; install it or remove the runtime`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

type missingErr struct{ bin string }

func (e *missingErr) Error() string { return e.bin + ": executable file not found" }

func TestValidateForNonoVersionMismatch(t *testing.T) {
	stubHost(t, found, func(bin, arg string) (string, error) { return "v22.1.0\n", nil })
	cfg := nonoCfg()
	cfg.Runtimes = map[string]string{"node": "20"}
	_, err := validateForNono(cfg)
	want := `runtime "node" pinned to 20 but host has 22.1.0`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

func TestValidateForNonoVersionMatch(t *testing.T) {
	stubHost(t, found, func(bin, arg string) (string, error) {
		if bin != "node" || arg != "--version" {
			t.Fatalf("probe = %s %s, want node --version", bin, arg)
		}
		return "v20.11.1\n", nil
	})
	cfg := nonoCfg()
	cfg.Runtimes = map[string]string{"node": "20"}
	if _, err := validateForNono(cfg); err != nil {
		t.Fatalf("validateForNono: %v", err)
	}
}

func TestValidateForNonoRustPinSkipsVersionCheck(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	cfg := nonoCfg()
	cfg.Runtimes = map[string]string{"rust": "1.79"}
	if _, err := validateForNono(cfg); err != nil {
		t.Fatalf("validateForNono: %v", err)
	}
}

func TestValidateForNonoUnpinnedSkipsVersionCheck(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	cfg := nonoCfg()
	cfg.Runtimes = map[string]string{"python": ""}
	if _, err := validateForNono(cfg); err != nil {
		t.Fatalf("validateForNono: %v", err)
	}
}

func TestValidateForNonoWarnings(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	cfg := nonoCfg()
	cfg.Claude.Theme = "dark"
	cfg.Shell.Aliases = map[string]string{"g": "git"}
	cfg.Memory = "some-kit"
	cfg.Scripts.OnStart = []string{"echo hi"}
	cfg.Php.Extensions = []string{"redis"}
	warnings, err := validateForNono(cfg)
	if err != nil {
		t.Fatalf("validateForNono: %v", err)
	}
	want := []string{
		"nono backend: ignoring 'claude:' settings (they would rewrite your real ~/.claude)",
		"nono backend: ignoring 'shell.aliases' (they would modify your real shell config)",
		"nono backend: ignoring 'memory:' (kits are sbx-only)",
		"nono backend: ignoring 'scripts.on_start' (nono has no per-session start hook)",
		"nono backend: ignoring 'php.extensions' (cannot install PHP extensions on the host)",
	}
	if len(warnings) != len(want) {
		t.Fatalf("warnings = %v, want %v", warnings, want)
	}
	for i := range want {
		if warnings[i] != want[i] {
			t.Errorf("warnings[%d] = %q, want %q", i, warnings[i], want[i])
		}
	}
}

func TestValidateForNonoCleanConfig(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	warnings, err := validateForNono(nonoCfg())
	if err != nil {
		t.Fatalf("validateForNono: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct{ in, want string }{
		{"v22.1.0\n", "22.1.0"},
		{"go version go1.24.1 darwin/arm64", "1.24.1"},
		{"Python 3.12.3", "3.12.3"},
		{"PHP 8.3.7 (cli) (built: ...)", "8.3.7"},
		{"10.2.0", "10.2.0"},
		{"no digits here", ""},
	}
	for _, tt := range tests {
		if got := extractVersion(tt.in); got != tt.want {
			t.Errorf("extractVersion(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestVersionMatches(t *testing.T) {
	tests := []struct {
		pin, host string
		want      bool
	}{
		{"20", "20.11.1", true},
		{"20", "22.1.0", false},
		{"1.24", "1.24.1", true},
		{"1.24", "1.23.0", false},
		{"3.12", "3.12", true},
		{"3.12.1", "3.12", false},
		{"20", "", false},
	}
	for _, tt := range tests {
		if got := versionMatches(tt.pin, tt.host); got != tt.want {
			t.Errorf("versionMatches(%q, %q) = %v, want %v", tt.pin, tt.host, got, tt.want)
		}
	}
}

func TestGenerateProfileFull(t *testing.T) {
	cfg := nonoCfg()
	cfg.Runtimes = map[string]string{"node": "20", "go": "", "bun": ""}
	cfg.Network = []string{"registry.npmjs.org"}
	cfg.Handoff = config.HandoffConfig{Enabled: true, URL: "http://localhost:8787"}
	cfg.MergedEnv = map[string]string{"FOO": "bar"}

	got, err := generateProfile(cfg)
	if err != nil {
		t.Fatalf("generateProfile: %v", err)
	}
	want := `// Generated by blvckhole from /proj/blvckhole.yaml. Do not edit — edit blvckhole.yaml and run 'blvckhole start'.
// Note: the 'workspace' key is ignored under nono; sessions run in the real project directory.
{
  "extends": [
    "claude-code"
  ],
  "meta": {
    "name": "myapp",
    "description": "Generated by blvckhole from /proj/blvckhole.yaml. Do not edit."
  },
  "groups": {
    "include": [
      "go_runtime",
      "node_runtime"
    ]
  },
  "workdir": {
    "access": "readwrite"
  },
  "filesystem": {
    "allow": [
      "$HOME/.bun"
    ]
  },
  "network": {
    "allow_domain": [
      "registry.npmjs.org"
    ],
    "open_port": [
      8787
    ]
  },
  "environment": {
    "set_vars": {
      "FOO": "bar"
    }
  }
}
`
	if string(got) != want {
		t.Errorf("generateProfile mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenerateProfileMinimalShellAgent(t *testing.T) {
	cfg := nonoCfg()
	cfg.Agent = "shell"
	got, err := generateProfile(cfg)
	if err != nil {
		t.Fatalf("generateProfile: %v", err)
	}
	s := string(got)
	if strings.Contains(s, `"extends"`) {
		t.Errorf("shell agent must not extend a preset:\n%s", s)
	}
	for _, absent := range []string{`"groups"`, `"filesystem"`, `"network"`, `"environment"`} {
		if strings.Contains(s, absent) {
			t.Errorf("minimal profile must omit %s:\n%s", absent, s)
		}
	}
	if !strings.Contains(s, `"access": "readwrite"`) {
		t.Errorf("profile must always grant workdir readwrite:\n%s", s)
	}
}

func TestGenerateProfileDeterministic(t *testing.T) {
	cfg := nonoCfg()
	cfg.Runtimes = map[string]string{"node": "", "pnpm": "", "python": "", "go": ""}
	cfg.MergedEnv = map[string]string{"B": "2", "A": "1", "C": "3"}
	first, err := generateProfile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		again, err := generateProfile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if string(again) != string(first) {
			t.Fatal("generateProfile output is not deterministic")
		}
	}
	// node and pnpm both map to node_runtime — it must appear once.
	if strings.Count(string(first), "node_runtime") != 1 {
		t.Errorf("node_runtime must be deduplicated:\n%s", first)
	}
}

func TestGenerateProfileBadHandoffPort(t *testing.T) {
	cfg := nonoCfg()
	cfg.Handoff = config.HandoffConfig{Enabled: true, URL: "http://localhost:notaport"}
	if _, err := generateProfile(cfg); err == nil {
		t.Fatal("want error for unparsable handoff port")
	}
}

func TestParseSessionsPlainArray(t *testing.T) {
	out := []byte(`[{"id":"s1","profile":"/proj/.config/blvckhole/nono/profile.jsonc"}]`)
	got, err := parseSessions(out)
	if err != nil {
		t.Fatalf("parseSessions: %v", err)
	}
	if len(got) != 1 || got[0].ID != "s1" {
		t.Fatalf("parseSessions = %+v", got)
	}
}

func TestParseSessionsWrappedObject(t *testing.T) {
	out := []byte(`{"sessions":[{"id":"s1","profile":"myapp"},{"id":"s2","profile":"other"}]}`)
	got, err := parseSessions(out)
	if err != nil {
		t.Fatalf("parseSessions: %v", err)
	}
	if len(got) != 2 || got[1].ID != "s2" {
		t.Fatalf("parseSessions = %+v", got)
	}
}

// A "sessions" key must win even when another key in the object (here
// "warnings") is also an empty JSON array — map iteration order is random,
// so picking the first key that happens to unmarshal as []nonoSession would
// nondeterministically return zero sessions.
func TestParseSessionsWrappedObjectSessionsKeyWinsOverEmptyArray(t *testing.T) {
	out := []byte(`{"warnings":[],"sessions":[{"id":"s1","profile":"myapp"}]}`)
	for i := 0; i < 20; i++ {
		got, err := parseSessions(out)
		if err != nil {
			t.Fatalf("parseSessions: %v", err)
		}
		if len(got) != 1 || got[0].ID != "s1" {
			t.Fatalf("parseSessions = %+v, want one session s1", got)
		}
	}
}

func TestParseSessionsWrappedObjectNoSessionArray(t *testing.T) {
	out := []byte(`{"warnings":["something"],"count":0}`)
	got, err := parseSessions(out)
	if err == nil {
		t.Fatalf("parseSessions = %+v, nil error; want error", got)
	}
	if !strings.Contains(err.Error(), "unexpected 'nono ps --json' output") {
		t.Fatalf("err = %q, want mention of unexpected output", err.Error())
	}
}

func TestParseSessionsEmptyOutput(t *testing.T) {
	for _, out := range []string{"", "  \n", "[]", "null"} {
		got, err := parseSessions([]byte(out))
		if err != nil {
			t.Fatalf("parseSessions(%q): %v", out, err)
		}
		if len(got) != 0 {
			t.Fatalf("parseSessions(%q) = %+v, want empty", out, got)
		}
	}
}

func TestParseSessionsInvalid(t *testing.T) {
	if _, err := parseSessions([]byte("not json")); err == nil {
		t.Fatal("parseSessions(invalid) = nil error, want error")
	}
}

func TestMatchSessions(t *testing.T) {
	sessions := []nonoSession{
		{ID: "s1", Profile: "/proj/.config/blvckhole/nono/profile.jsonc"},
		{ID: "s2", Profile: "myapp"},
		{ID: "s3", Profile: "/other/profile.jsonc"},
		{ID: "s4", Profile: "otherapp"},
	}
	got := matchSessions(sessions, "/proj/.config/blvckhole/nono/profile.jsonc", "myapp")
	if len(got) != 2 || got[0].ID != "s1" || got[1].ID != "s2" {
		t.Fatalf("matchSessions = %+v, want s1 and s2", got)
	}
	if got := matchSessions(sessions, "/nope", "nope"); len(got) != 0 {
		t.Fatalf("matchSessions(no match) = %+v, want empty", got)
	}
}

func TestNonoName(t *testing.T) {
	n := &NonoBackend{}
	if n.Name() != "nono" {
		t.Fatalf("Name() = %q, want nono", n.Name())
	}
}

func TestNonoEnsureAvailableMissing(t *testing.T) {
	t.Setenv("PATH", "")
	err := (&NonoBackend{}).EnsureAvailable()
	if err == nil {
		t.Fatal("EnsureAvailable with empty PATH = nil, want error")
	}
	want := "nono is not installed. Install it from: https://nono.sh"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestNonoExistsAndHashRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	n := &NonoBackend{}

	if n.Exists(cfg) {
		t.Fatal("Exists = true before any provisioning")
	}
	if _, err := n.ReadConfigHash(cfg); err == nil {
		t.Fatal("ReadConfigHash = nil error with no hash file, want error")
	}

	if err := n.WriteConfigHash(cfg, "abc123"); err != nil {
		t.Fatalf("WriteConfigHash: %v", err)
	}
	got, err := n.ReadConfigHash(cfg)
	if err != nil {
		t.Fatalf("ReadConfigHash: %v", err)
	}
	if got != "abc123" {
		t.Fatalf("ReadConfigHash = %q, want abc123", got)
	}

	// Exists keys off the profile file, not the hash sidecar.
	if n.Exists(cfg) {
		t.Fatal("Exists = true without profile.jsonc")
	}
	profile := filepath.Join(dir, ".config", "blvckhole", "nono", "profile.jsonc")
	if err := os.WriteFile(profile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if !n.Exists(cfg) {
		t.Fatal("Exists = false with profile.jsonc present")
	}
}

// stubNono puts a fake `nono` binary on PATH whose `ps --json` prints an
// empty session list, so lifecycle methods can run without a real install.
func stubNono(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	script := "#!/bin/sh\necho '[]'\n"
	if err := os.WriteFile(filepath.Join(binDir, "nono"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
}

func TestNonoRemoveDeletesState(t *testing.T) {
	stubNono(t)
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	n := &NonoBackend{}
	if err := n.WriteConfigHash(cfg, "abc"); err != nil {
		t.Fatal(err)
	}
	if err := n.Remove(cfg); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".config", "blvckhole", "nono")); !os.IsNotExist(err) {
		t.Fatal("Remove must delete the nono state directory")
	}
}

func TestNonoIsRunningNoSessions(t *testing.T) {
	stubNono(t)
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	n := &NonoBackend{}
	profile := filepath.Join(dir, ".config", "blvckhole", "nono", "profile.jsonc")
	if err := os.MkdirAll(filepath.Dir(profile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if n.IsRunning(cfg) {
		t.Fatal("IsRunning = true with no live sessions")
	}
	info, err := n.Status(cfg)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if info == nil || info.Status != "stopped" || info.Name != "myapp" {
		t.Fatalf("Status = %+v, want stopped myapp", info)
	}
	if len(info.Workspaces) != 1 || info.Workspaces[0] != dir {
		t.Fatalf("Workspaces = %v, want [%s]", info.Workspaces, dir)
	}
}

func TestNonoNetworkMutationErrors(t *testing.T) {
	cfg := nonoCfg()
	n := &NonoBackend{}
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{"allow", func() error { return n.AllowNetwork(cfg, []string{"example.com"}) },
			"nono applies network policy when a session launches and cannot change it at runtime. Add the domain to 'network:' in /proj/blvckhole.yaml and run 'blvckhole restart'."},
		{"deny", func() error { return n.DenyNetwork(cfg, []string{"example.com"}) },
			"nono applies network policy when a session launches and cannot change it at runtime. nono blocks every domain not allowed; remove it from 'network:' in /proj/blvckhole.yaml and run 'blvckhole restart'."},
		{"remove", func() error { return n.RemoveNetwork(cfg, []string{"example.com"}) },
			"nono applies network policy when a session launches and cannot change it at runtime. Remove the domain from 'network:' in /proj/blvckhole.yaml and run 'blvckhole restart'."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil || err.Error() != tt.want {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestNonoPrepareAgentIsNoOp(t *testing.T) {
	if err := (&NonoBackend{}).PrepareAgent(nonoCfg()); err != nil {
		t.Fatalf("PrepareAgent = %v, want nil", err)
	}
}

func TestGetNono(t *testing.T) {
	b := Get("nono")
	if b == nil {
		t.Fatal(`Get("nono") = nil, want NonoBackend`)
	}
	if b.Name() != "nono" {
		t.Fatalf("Name() = %q, want nono", b.Name())
	}
}

func TestNonoProvisionValidationErrorWritesNothing(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	cfg.Packages = []string{"jq"}
	n := &NonoBackend{}
	if err := n.Provision(cfg); err == nil {
		t.Fatal("Provision with packages must fail validation")
	}
	if _, err := os.Stat(filepath.Join(dir, ".config", "blvckhole", "nono")); !os.IsNotExist(err) {
		t.Fatal("Provision must not create profile state when validation fails")
	}
}

func TestNonoProvisionValidationErrorPreservesExistingProfile(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	n := &NonoBackend{}
	profileDir := filepath.Join(dir, ".config", "blvckhole", "nono")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "profile.jsonc"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg.Packages = []string{"jq"} // config edited into an invalid state
	if err := n.Provision(cfg); err == nil {
		t.Fatal("Provision with packages must fail validation")
	}
	if _, err := os.Stat(filepath.Join(profileDir, "profile.jsonc")); err != nil {
		t.Fatal("Provision must not destroy the last good profile when the new config fails validation")
	}
}

// stubNonoValidateFails puts a fake `nono` on PATH whose `profile validate`
// subcommand fails; every other subcommand (e.g. `ps --json`) succeeds with
// an empty session list.
func stubNonoValidateFails(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	script := "#!/bin/sh\ncase \"$1\" in\n  profile) exit 1 ;;\n  *) echo '[]' ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "nono"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
}

// stubNonoOnCreateFails puts a fake `nono` on PATH whose `profile validate`
// succeeds but `run` (used to execute on_create commands) fails.
func stubNonoOnCreateFails(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	script := "#!/bin/sh\ncase \"$1\" in\n  profile) exit 0 ;;\n  run) exit 1 ;;\n  *) echo '[]' ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "nono"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
}

func TestNonoProvisionValidateFailureRemovesProfile(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	stubNonoValidateFails(t)
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	n := &NonoBackend{}

	if err := n.Provision(cfg); err == nil {
		t.Fatal("Provision must fail when nono profile validate fails")
	}
	if n.Exists(cfg) {
		t.Fatal("Exists = true after failed validate; the broken profile must be removed")
	}
	if _, err := os.Stat(n.profileDir(cfg)); !os.IsNotExist(err) {
		t.Fatal("profile directory must be removed after a failed validate")
	}
}

func TestNonoProvisionOnCreateFailureRemovesProfile(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	stubNonoOnCreateFails(t)
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	cfg.Scripts.OnCreate = []string{"echo hi"}
	n := &NonoBackend{}

	if err := n.Provision(cfg); err == nil {
		t.Fatal("Provision must fail when an on_create command fails")
	}
	if n.Exists(cfg) {
		t.Fatal("Exists = true after failed on_create; the broken profile must be removed")
	}
	if _, err := os.Stat(n.profileDir(cfg)); !os.IsNotExist(err) {
		t.Fatal("profile directory must be removed after a failed on_create command")
	}
}

func TestNonoProvisionFastPathWhenHashMatches(t *testing.T) {
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	n := &NonoBackend{}

	// Seed an existing profile whose stored hash matches the current
	// config file — mimics an idle nono sandbox left over from a prior
	// successful run.
	if err := os.MkdirAll(n.profileDir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(n.profilePath(cfg), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg.ConfigPath = filepath.Join(dir, "blvckhole.yaml")
	if err := os.WriteFile(cfg.ConfigPath, []byte("name: myapp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	hash := cfg.FileHash()
	if hash == "" {
		t.Fatal("FileHash() = \"\", want non-empty")
	}
	if err := n.WriteConfigHash(cfg, hash); err != nil {
		t.Fatal(err)
	}

	// Deliberately no `nono` on PATH: if Provision took the slow
	// (destroy-and-recreate) path it would try to exec `nono profile
	// validate` and fail, so a passing Provision proves the fast path.
	t.Setenv("PATH", "")

	if err := n.Provision(cfg); err != nil {
		t.Fatalf("Provision with a matching config hash must be a cheap no-op, got: %v", err)
	}
	got, err := os.ReadFile(n.profilePath(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{}" {
		t.Fatal("Provision must not rewrite an up-to-date profile")
	}
}

func TestNonoProvisionMismatchedHashReprovisions(t *testing.T) {
	stubHost(t, found, noVersionCalls(t))
	stubNono(t)
	dir := t.TempDir()
	cfg := nonoCfg()
	cfg.ProjectDir = dir
	cfg.ConfigPath = filepath.Join(dir, "blvckhole.yaml")
	if err := os.WriteFile(cfg.ConfigPath, []byte("name: myapp\n"), 0644); err != nil {
		t.Fatal(err)
	}
	n := &NonoBackend{}

	if err := os.MkdirAll(n.profileDir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(n.profilePath(cfg), []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := n.WriteConfigHash(cfg, "not-the-real-hash"); err != nil {
		t.Fatal(err)
	}

	if err := n.Provision(cfg); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	got, err := os.ReadFile(n.profilePath(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == "stale" {
		t.Fatal("Provision must regenerate the profile when the stored hash does not match the current config")
	}
	newHash, err := n.ReadConfigHash(cfg)
	if err != nil {
		t.Fatalf("ReadConfigHash after reprovision: %v", err)
	}
	if newHash != cfg.FileHash() {
		t.Fatal("Provision must store the current config hash after reprovisioning")
	}
}
