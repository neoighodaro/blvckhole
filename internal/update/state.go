package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State is the cached result of the last background update check.
type State struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
}

// StatePath returns the path to the update-state cache file.
func StatePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "blvckhole", "update.json"), nil
}

// LoadState reads the cache file. A missing or corrupt file yields a zero
// State and no error, so callers can treat "never checked" uniformly.
func LoadState(path string) State {
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}
	}
	return s
}

// SaveState writes the cache file, creating parent directories as needed.
func SaveState(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Due reports whether a fresh check should run: true when no prior check is
// recorded or the last check is older than interval.
func (s State) Due(now time.Time, interval time.Duration) bool {
	if s.LastChecked.IsZero() {
		return true
	}
	return now.Sub(s.LastChecked) >= interval
}
