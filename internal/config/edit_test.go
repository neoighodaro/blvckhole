package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "blvckhole.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}

func TestAddNetworkDomainsExisting(t *testing.T) {
	path := writeTemp(t, "name: demo\n# allowlist\nnetwork:\n  - api.github.com\n")
	if err := AddNetworkDomains(path, []string{"elevenlabs.io"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	out := readFile(t, path)
	if !strings.Contains(out, "api.github.com") || !strings.Contains(out, "elevenlabs.io") {
		t.Fatalf("missing domains:\n%s", out)
	}
	if !strings.Contains(out, "# allowlist") {
		t.Fatalf("comment not preserved:\n%s", out)
	}
	if !strings.Contains(out, "name: demo") {
		t.Fatalf("other keys lost:\n%s", out)
	}
}

func TestAddNetworkDomainsAbsentKey(t *testing.T) {
	path := writeTemp(t, "name: demo\n")
	if err := AddNetworkDomains(path, []string{"elevenlabs.io"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	out := readFile(t, path)
	if !strings.Contains(out, "network:") || !strings.Contains(out, "elevenlabs.io") {
		t.Fatalf("network key not created:\n%s", out)
	}
}

func TestAddNetworkDomainsEmptySequence(t *testing.T) {
	path := writeTemp(t, "name: demo\nnetwork: []\n")
	if err := AddNetworkDomains(path, []string{"elevenlabs.io"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	out := readFile(t, path)
	if !strings.Contains(out, "elevenlabs.io") {
		t.Fatalf("not added to empty list:\n%s", out)
	}
}

func TestAddNetworkDomainsDuplicate(t *testing.T) {
	path := writeTemp(t, "network:\n  - api.github.com\n")
	if err := AddNetworkDomains(path, []string{"api.github.com"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	out := readFile(t, path)
	if strings.Count(out, "api.github.com") != 1 {
		t.Fatalf("duplicate added:\n%s", out)
	}
}

func TestRemoveNetworkDomains(t *testing.T) {
	path := writeTemp(t, "network:\n  - api.github.com\n  - elevenlabs.io\n")
	if err := RemoveNetworkDomains(path, []string{"elevenlabs.io"}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	out := readFile(t, path)
	if strings.Contains(out, "elevenlabs.io") {
		t.Fatalf("domain not removed:\n%s", out)
	}
	if !strings.Contains(out, "api.github.com") {
		t.Fatalf("wrong domain removed:\n%s", out)
	}
}

func TestRemoveNetworkDomainsAbsent(t *testing.T) {
	path := writeTemp(t, "network:\n  - api.github.com\n")
	if err := RemoveNetworkDomains(path, []string{"nope.example.com"}); err != nil {
		t.Fatalf("remove absent should be no-op: %v", err)
	}
	out := readFile(t, path)
	if !strings.Contains(out, "api.github.com") {
		t.Fatalf("unrelated domain lost:\n%s", out)
	}
}

func TestAddNetworkRoundTripParses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blvckhole.yaml")
	if err := os.WriteFile(path, []byte("name: demo\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := AddNetworkDomains(path, []string{"a.com", "b.com"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	cfg, err := Parse(path, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cfg.Network) != 2 || cfg.Network[0] != "a.com" || cfg.Network[1] != "b.com" {
		t.Fatalf("round-trip mismatch: %#v", cfg.Network)
	}
}
