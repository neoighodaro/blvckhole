package update

import "testing"

func TestIsRelease(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"v0.0.5", true},
		{"v1.2.3", true},
		{"dev", false},
		{"", false},
		{"1.2.3", false}, // missing leading v
	}
	for _, c := range cases {
		if got := IsRelease(c.in); got != c.want {
			t.Errorf("IsRelease(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.0.4", "v0.0.5", true},
		{"v0.0.5", "v0.0.5", false},
		{"v0.0.6", "v0.0.5", false},
		{"dev", "v0.0.5", false},     // invalid current
		{"v0.0.4", "garbage", false}, // invalid latest
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q,%q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}
