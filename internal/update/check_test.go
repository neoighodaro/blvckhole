package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestSuppressed(t *testing.T) {
	none := func(string) string { return "" }

	cases := []struct {
		name    string
		getenv  func(string) string
		tty     bool
		command string
		want    bool
	}{
		{"normal interactive", none, true, "status", false},
		{"opt-out env", func(k string) string {
			if k == "BLVCKHOLE_NO_UPDATE_CHECK" {
				return "1"
			}
			return ""
		}, true, "status", true},
		{"sandbox", func(k string) string {
			if k == "IS_SANDBOX" {
				return "1"
			}
			return ""
		}, true, "status", true},
		{"ci", func(k string) string {
			if k == "CI" {
				return "true"
			}
			return ""
		}, true, "status", true},
		{"not a tty", none, false, "status", true},
		{"update command", none, true, "update", true},
		{"check command", none, true, "__update-check", true},
		{"version command", none, true, "version", true},
	}
	for _, c := range cases {
		if got := Suppressed(c.getenv, c.tty, c.command); got != c.want {
			t.Errorf("%s: Suppressed = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRunCheckSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name": "v0.0.9"}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	path := filepath.Join(t.TempDir(), "update.json")
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)

	if err := RunCheck(context.Background(), c, path, now); err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	got := LoadState(path)
	if got.LatestVersion != "v0.0.9" {
		t.Errorf("LatestVersion = %q, want v0.0.9", got.LatestVersion)
	}
	if !got.LastChecked.Equal(now) {
		t.Errorf("LastChecked = %v, want %v", got.LastChecked, now)
	}
}

func TestRunCheckFailureStillThrottles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	path := filepath.Join(t.TempDir(), "update.json")
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)

	if err := RunCheck(context.Background(), c, path, now); err == nil {
		t.Error("RunCheck should return the fetch error")
	}
	// LastChecked must still be written so failures don't cause retry storms.
	got := LoadState(path)
	if !got.LastChecked.Equal(now) {
		t.Errorf("LastChecked = %v, want %v (throttle on failure)", got.LastChecked, now)
	}
	if got.LatestVersion != "" {
		t.Errorf("LatestVersion = %q, want empty on failure", got.LatestVersion)
	}
}
