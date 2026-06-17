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
