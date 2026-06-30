// Package update implements background update checks and self-update.
package update

import "golang.org/x/mod/semver"

// IsRelease reports whether v is a valid release version tag (e.g. "v1.2.3").
func IsRelease(v string) bool {
	return semver.IsValid(v)
}

// IsNewer reports whether latest is strictly newer than current.
// It returns false if either value is not a valid semver tag.
func IsNewer(current, latest string) bool {
	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return false
	}
	return semver.Compare(latest, current) > 0
}
