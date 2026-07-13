package backend

import "testing"

func TestGetUnknownReturnsNil(t *testing.T) {
	if b := Get("does-not-exist"); b != nil {
		t.Fatalf("Get(unknown) = %v, want nil", b)
	}
}

func TestNamesSorted(t *testing.T) {
	got := Names()
	want := []string{"nono", "sbx"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/plain/dir", "'/plain/dir'"},
		{"/it's here", `'/it'\''s here'`},
		{"", "''"},
	}
	for _, tt := range tests {
		if got := shellQuote(tt.in); got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
