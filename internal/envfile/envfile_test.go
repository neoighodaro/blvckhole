package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_BasicKeyValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("FOO=bar\nBAZ=qux\n"), 0644)

	got, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", got["FOO"], "bar")
	}
	if got["BAZ"] != "qux" {
		t.Errorf("BAZ = %q, want %q", got["BAZ"], "qux")
	}
}

func TestParse_CommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("# comment\n\nKEY=value\n  # indented comment\n"), 0644)

	got, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d entries, want 1", len(got))
	}
	if got["KEY"] != "value" {
		t.Errorf("KEY = %q, want %q", got["KEY"], "value")
	}
}

func TestParse_QuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("A=\"hello world\"\nB='single quotes'\n"), 0644)

	got, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["A"] != "hello world" {
		t.Errorf("A = %q, want %q", got["A"], "hello world")
	}
	if got["B"] != "single quotes" {
		t.Errorf("B = %q, want %q", got["B"], "single quotes")
	}
}

func TestParse_ValueWithEquals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("URL=https://example.com?a=1&b=2\n"), 0644)

	got, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["URL"] != "https://example.com?a=1&b=2" {
		t.Errorf("URL = %q, want %q", got["URL"], "https://example.com?a=1&b=2")
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
