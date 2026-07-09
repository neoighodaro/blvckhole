package backend

import "testing"

func TestGetUnknownReturnsNil(t *testing.T) {
	if b := Get("does-not-exist"); b != nil {
		t.Fatalf("Get(unknown) = %v, want nil", b)
	}
}

func TestNamesContainsSbx(t *testing.T) {
	got := Names()
	if len(got) != 1 || got[0] != "sbx" {
		t.Fatalf("Names() = %v, want [sbx]", got)
	}
}
