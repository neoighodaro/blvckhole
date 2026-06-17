package handoff

import (
	"regexp"
	"testing"
)

func TestNewID_Format(t *testing.T) {
	id := NewID()
	if len(id) != 24 {
		t.Fatalf("len(NewID()) = %d, want 24", len(id))
	}
	if !regexp.MustCompile(`^[0-9a-f]{24}$`).MatchString(id) {
		t.Errorf("NewID() = %q, want 24 lowercase hex chars", id)
	}
}

func TestNewID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := NewID()
		if seen[id] {
			t.Fatalf("duplicate id generated: %q", id)
		}
		seen[id] = true
	}
}
