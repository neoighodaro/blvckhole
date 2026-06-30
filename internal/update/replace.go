package update

import (
	"os"
	"path/filepath"
)

// ReplaceBinary atomically replaces target with binary. The temp file is
// created in target's directory so the final rename stays on one filesystem.
// Replacing a running executable is safe on Unix: the old inode survives until
// the running process exits.
func ReplaceBinary(target string, binary []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".blvckhole-update-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed; cleans up on failure

	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0755); err != nil {
		return err
	}
	return os.Rename(tmpName, target)
}
