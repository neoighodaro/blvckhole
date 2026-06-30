package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "update.json")
	want := State{
		LastChecked:   time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC),
		LatestVersion: "v0.0.5",
	}
	if err := SaveState(path, want); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got := LoadState(path)
	if !got.LastChecked.Equal(want.LastChecked) || got.LatestVersion != want.LatestVersion {
		t.Errorf("round-trip = %+v, want %+v", got, want)
	}
}

func TestLoadMissingOrCorrupt(t *testing.T) {
	missing := LoadState(filepath.Join(t.TempDir(), "nope.json"))
	if !missing.LastChecked.IsZero() || missing.LatestVersion != "" {
		t.Errorf("missing file = %+v, want zero State", missing)
	}

	corrupt := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(corrupt, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := LoadState(corrupt); !got.LastChecked.IsZero() {
		t.Errorf("corrupt file = %+v, want zero State", got)
	}
}

func TestDue(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	interval := 24 * time.Hour

	if !(State{}).Due(now, interval) {
		t.Error("zero LastChecked should be Due")
	}
	fresh := State{LastChecked: now.Add(-1 * time.Hour)}
	if fresh.Due(now, interval) {
		t.Error("1h-old check should not be Due with 24h interval")
	}
	stale := State{LastChecked: now.Add(-25 * time.Hour)}
	if !stale.Due(now, interval) {
		t.Error("25h-old check should be Due with 24h interval")
	}
}
