package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "blvckhole.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBackendDefaultsToSbx(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "name: myapp\n")
	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backend != "sbx" {
		t.Fatalf("Backend = %q, want \"sbx\"", cfg.Backend)
	}
}

func TestBackendExplicitValueKept(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, "name: myapp\nbackend: nono\n")
	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backend != "nono" {
		t.Fatalf("Backend = %q, want \"nono\"", cfg.Backend)
	}
}

func TestFileHash(t *testing.T) {
	dir := t.TempDir()
	content := "name: myapp\n"
	path := writeConfig(t, dir, content)
	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(content))
	want := hex.EncodeToString(sum[:])
	if got := cfg.FileHash(); got != want {
		t.Fatalf("FileHash() = %q, want %q", got, want)
	}
}

func TestFileHashMissingFile(t *testing.T) {
	cfg := &Config{ConfigPath: "/nonexistent/blvckhole.yaml"}
	if got := cfg.FileHash(); got != "" {
		t.Fatalf("FileHash() = %q, want \"\"", got)
	}
}
