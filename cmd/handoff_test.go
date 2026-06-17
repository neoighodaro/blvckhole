package cmd

import "testing"

func TestHandoffCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "handoff" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("handoff command not registered on rootCmd")
	}
}

func TestHandoffFlagDefaults(t *testing.T) {
	if got := handoffCmd.Flags().Lookup("port").DefValue; got != "8787" {
		t.Errorf("--port default = %q, want 8787", got)
	}
	if got := handoffCmd.Flags().Lookup("bind").DefValue; got != "127.0.0.1" {
		t.Errorf("--bind default = %q, want 127.0.0.1", got)
	}
	if got := handoffCmd.Flags().Lookup("store").DefValue; got != "" {
		t.Errorf("--store default = %q, want empty", got)
	}
}

func TestHandoffReplaceFlag(t *testing.T) {
	f := handoffCmd.Flags().Lookup("replace")
	if f == nil {
		t.Fatal("--replace flag not registered")
	}
	if f.Shorthand != "R" {
		t.Errorf("--replace shorthand = %q, want R", f.Shorthand)
	}
	if f.DefValue != "false" {
		t.Errorf("--replace default = %q, want false", f.DefValue)
	}
}
