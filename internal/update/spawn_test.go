package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSpawnDetachedRuns verifies Spawn starts a process that survives the
// parent returning. We point it at a fake executable that, when given the
// "__update-check" argument, touches a marker file.
func TestSpawnDetachedRuns(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	script := filepath.Join(dir, "fake")
	// #!/bin/sh — write the marker only for the __update-check subcommand.
	body := "#!/bin/sh\nif [ \"$1\" = \"__update-check\" ]; then : > \"" + marker + "\"; fi\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}

	if err := Spawn(script); err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// Detached child runs asynchronously; poll briefly for the marker.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return // success
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("spawned process did not create marker file")
}
