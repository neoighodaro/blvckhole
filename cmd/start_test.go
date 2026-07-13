package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neoighodaro/blvckhole/internal/backend"
	"github.com/neoighodaro/blvckhole/internal/config"
)

// fakeStartBackend is a minimal backend.Backend that records which lifecycle
// methods runStart calls, without shelling out to a real sbx/nono install.
// Provisioning ownership of "stale sandbox" cleanup moved from runStart into
// each backend's Provision (nono: idle sessions are normal, not stale; a
// blind Remove+Provision from runStart would destroy a live profile and
// re-run on_create every time). This fake proves runStart no longer removes
// anything itself.
type fakeStartBackend struct {
	exists     bool
	running    bool
	configHash string

	removeCalled    bool
	provisionCalled bool
}

var _ backend.Backend = (*fakeStartBackend)(nil)

func (f *fakeStartBackend) Name() string                                 { return "fake" }
func (f *fakeStartBackend) EnsureAvailable() error                       { return nil }
func (f *fakeStartBackend) Exists(*config.Config) bool                   { return f.exists }
func (f *fakeStartBackend) IsRunning(*config.Config) bool                { return f.running }
func (f *fakeStartBackend) Status(*config.Config) (*backend.Info, error) { return nil, nil }

func (f *fakeStartBackend) Provision(*config.Config) error {
	f.provisionCalled = true
	return nil
}

func (f *fakeStartBackend) Run(*config.Config, ...string) error                  { return nil }
func (f *fakeStartBackend) RunCommand(*config.Config, string) error              { return nil }
func (f *fakeStartBackend) ExecSilent(*config.Config, ...string) (string, error) { return "", nil }
func (f *fakeStartBackend) Shell(*config.Config, string) error                   { return nil }
func (f *fakeStartBackend) PrepareAgent(*config.Config) error                    { return nil }
func (f *fakeStartBackend) Stop(*config.Config) error                            { return nil }

func (f *fakeStartBackend) Remove(*config.Config) error {
	f.removeCalled = true
	return nil
}

func (f *fakeStartBackend) AllowNetwork(*config.Config, []string) error  { return nil }
func (f *fakeStartBackend) DenyNetwork(*config.Config, []string) error   { return nil }
func (f *fakeStartBackend) RemoveNetwork(*config.Config, []string) error { return nil }

func (f *fakeStartBackend) ReadConfigHash(*config.Config) (string, error) {
	if f.configHash == "" {
		return "", os.ErrNotExist
	}
	return f.configHash, nil
}

func (f *fakeStartBackend) WriteConfigHash(*config.Config, string) error { return nil }

// parsedTestConfig writes a real blvckhole.yaml to a temp dir and parses it
// via config.Parse, so cfg.FileHash() reflects real file bytes on disk (as
// runStart/configChanged require).
func parsedTestConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "blvckhole.yaml")
	if err := os.WriteFile(path, []byte("name: myapp\nbackend: nono\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Parse(path, dir)
	if err != nil {
		t.Fatalf("config.Parse: %v", err)
	}
	return cfg
}

func TestRunStartIdleExistingSandboxDoesNotRemove(t *testing.T) {
	cfg := parsedTestConfig(t)
	hash := cfg.FileHash()
	if hash == "" {
		t.Fatal("FileHash() = \"\", want non-empty")
	}

	b := &fakeStartBackend{exists: true, running: false, configHash: hash}
	if err := runStart(b, cfg); err != nil {
		t.Fatalf("runStart: %v", err)
	}
	if b.removeCalled {
		t.Fatal("runStart must not call Remove on an idle (Exists, !IsRunning) sandbox — that decision belongs to Provision")
	}
	if !b.provisionCalled {
		t.Fatal("runStart must call Provision on an idle sandbox")
	}
}

func TestRunStartIdleExistingSandboxHashMismatchStillDoesNotRemove(t *testing.T) {
	cfg := parsedTestConfig(t)

	b := &fakeStartBackend{exists: true, running: false, configHash: "stale-hash-that-does-not-match"}
	if err := runStart(b, cfg); err != nil {
		t.Fatalf("runStart: %v", err)
	}
	if b.removeCalled {
		t.Fatal("runStart must not call Remove even when the stored config hash does not match — removal on config drift is the backend's job now")
	}
	if !b.provisionCalled {
		t.Fatal("runStart must call Provision on an idle sandbox")
	}
}
