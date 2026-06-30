package update

import (
	"context"
	"time"
)

// CheckInterval is the minimum time between background network checks.
const CheckInterval = 24 * time.Hour

// Suppressed reports whether the background update check should be skipped
// entirely (no spawn, no notice).
func Suppressed(getenv func(string) string, stdoutIsTTY bool, command string) bool {
	if getenv("BLVCKHOLE_NO_UPDATE_CHECK") != "" {
		return true
	}
	if getenv("IS_SANDBOX") != "" {
		return true
	}
	if getenv("CI") != "" {
		return true
	}
	if !stdoutIsTTY {
		return true
	}
	switch command {
	case "update", "__update-check", "version":
		return true
	}
	return false
}

// RunCheck fetches the latest release and updates the state cache. LastChecked
// is always set (so failures still throttle); LatestVersion is set only on a
// successful fetch. The returned error is for callers that care; the detached
// background child ignores it.
func RunCheck(ctx context.Context, c *Client, statePath string, now time.Time) error {
	s := LoadState(statePath)
	s.LastChecked = now

	rel, err := c.LatestRelease(ctx)
	if err != nil {
		_ = SaveState(statePath, s)
		return err
	}
	s.LatestVersion = rel.TagName
	return SaveState(statePath, s)
}
