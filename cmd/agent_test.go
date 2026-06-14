package cmd

import "testing"

func TestClientsFocusPane(t *testing.T) {
	const output = `CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND
1         terminal_5     zellij action list-clients`

	tests := []struct {
		name   string
		paneID string
		want   bool
	}{
		{"focused pane matches", "5", true},
		{"different pane", "4", false},
		{"empty pane id never matches", "", false},
		{"no substring false positive", "terminal_5", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientsFocusPane(output, tt.paneID); got != tt.want {
				t.Errorf("clientsFocusPane(paneID=%q) = %v, want %v", tt.paneID, got, tt.want)
			}
		})
	}
}

// Real `zellij action list-tabs --state --json` output: Tab #3 is active with
// no floating panes; "Yulo API" is inactive but has floating panes visible.
const listTabsJSON = `[
  {"position":0,"name":" Yulo API","active":false,"are_floating_panes_visible":true,"tab_id":0},
  {"position":1,"name":" Yulo Web","active":false,"are_floating_panes_visible":false,"tab_id":1},
  {"position":2,"name":"Tab #3","active":true,"are_floating_panes_visible":false,"tab_id":2}
]`

func TestActiveZellijTab(t *testing.T) {
	active, ok := activeZellijTab(listTabsJSON)
	if !ok {
		t.Fatal("activeZellijTab() ok = false, want true")
	}
	if active.Name != "Tab #3" {
		t.Errorf("active.Name = %q, want %q", active.Name, "Tab #3")
	}
	if active.FloatingVisible {
		t.Errorf("active.FloatingVisible = true, want false")
	}
}

func TestActiveZellijTabFloatingActive(t *testing.T) {
	const j = `[
  {"name":" Yulo API","active":true,"are_floating_panes_visible":true},
  {"name":"Tab #3","active":false,"are_floating_panes_visible":false}
]`
	active, ok := activeZellijTab(j)
	if !ok {
		t.Fatal("activeZellijTab() ok = false, want true")
	}
	if !active.FloatingVisible {
		t.Error("active.FloatingVisible = false, want true")
	}
}

func TestActiveZellijTabNoActiveOrBadInput(t *testing.T) {
	if _, ok := activeZellijTab(`[{"name":"a","active":false}]`); ok {
		t.Error("activeZellijTab() ok = true for no active tab, want false")
	}
	if _, ok := activeZellijTab("not json"); ok {
		t.Error("activeZellijTab() ok = true for bad input, want false")
	}
}
