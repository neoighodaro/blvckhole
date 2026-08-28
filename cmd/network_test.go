package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestValidateTimedAllow(t *testing.T) {
	tests := []struct {
		name    string
		mins    int
		persist bool
		wantErr bool
	}{
		{"disabled", 0, false, false},
		{"disabled with persist", 0, true, false},
		{"min valid", 1, false, false},
		{"max valid", 30, false, false},
		{"mid valid", 10, false, false},
		{"zero-with-mins is off", 0, false, false},
		{"below range", -5, false, true},
		{"above range", 31, false, true},
		{"with persist", 10, true, true},
		{"with persist at boundary", 30, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimedAllow(tt.mins, tt.persist)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateTimedAllow(%d, %v) error = %v, wantErr %v", tt.mins, tt.persist, err, tt.wantErr)
			}
		})
	}
}

func TestFormatRemaining(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{10 * time.Minute, "10:00"},
		{9*time.Minute + 59*time.Second, "9:59"},
		{372 * time.Second, "6:12"},
		{500 * time.Millisecond, "0:01"}, // rounds up to next second
		{0, "0:00"},
		{-3 * time.Second, "0:00"}, // never negative
		{61 * time.Second, "1:01"},
	}
	for _, tt := range tests {
		if got := formatRemaining(tt.d); got != tt.want {
			t.Errorf("formatRemaining(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestCountdownBar(t *testing.T) {
	total := 10 * time.Minute
	width := 10

	if got := countdownBar(0, total, width); got != strings.Repeat("░", 10) {
		t.Errorf("empty bar = %q", got)
	}
	if got := countdownBar(total, total, width); got != strings.Repeat("█", 10) {
		t.Errorf("full bar = %q", got)
	}
	if got := countdownBar(5*time.Minute, total, width); got != strings.Repeat("█", 5)+strings.Repeat("░", 5) {
		t.Errorf("half bar = %q", got)
	}

	// Always exactly width runes wide, and clamped past the ends.
	for _, elapsed := range []time.Duration{-time.Minute, 0, 3 * time.Minute, total, total + time.Minute} {
		got := countdownBar(elapsed, total, width)
		if n := len([]rune(got)); n != width {
			t.Errorf("countdownBar width for elapsed=%v = %d, want %d (%q)", elapsed, n, width, got)
		}
	}
}

func TestRenderCountdownLine(t *testing.T) {
	line := renderCountdownLine("example.com", 4*time.Minute, 10*time.Minute, 8)
	if !strings.Contains(line, "example.com") {
		t.Errorf("line missing label: %q", line)
	}
	if !strings.Contains(line, "6:00 remaining") {
		t.Errorf("line missing remaining time: %q", line)
	}
	if !strings.Contains(line, "[") || !strings.Contains(line, "]") {
		t.Errorf("line missing bar brackets: %q", line)
	}
}
