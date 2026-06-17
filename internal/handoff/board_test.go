package handoff

import (
	"strings"
	"testing"
)

func TestRenderBoard_Empty(t *testing.T) {
	var b strings.Builder
	if err := RenderBoard(&b, nil, "mission"); err != nil {
		t.Fatalf("RenderBoard error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "No threads yet") {
		t.Error("empty board should say 'No threads yet'")
	}
	if !strings.Contains(out, `http-equiv="refresh"`) {
		t.Error("board should have a meta refresh")
	}
}

func TestRenderBoard_RendersThreadAndEscapes(t *testing.T) {
	threads := []Thread{
		{
			ID: "1", From: "api", To: "web", Subject: "Hi <script>", Status: StatusOpen,
			Messages: []Message{{From: "api", Body: "body & <b>", At: "2026-06-17T10:00:00Z"}},
		},
	}
	var b strings.Builder
	if err := RenderBoard(&b, threads, "mission"); err != nil {
		t.Fatalf("RenderBoard error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "api") || !strings.Contains(out, "web") {
		t.Error("board should show from and to")
	}
	if strings.Contains(out, "Hi <script>") {
		t.Error("subject must be HTML-escaped, found raw <script>")
	}
	if !strings.Contains(out, "Hi &lt;script&gt;") {
		t.Error("subject should appear escaped")
	}
	if !strings.Contains(out, "body &amp; &lt;b&gt;") {
		t.Error("message body should appear escaped")
	}
}

// TestRenderBoard_AnsweredCollapsedByDefault verifies the mission board renders
// each thread as a collapsible <details>: open threads start expanded (the open
// attribute is present) while answered threads start collapsed (no open
// attribute). It also checks the state-persistence script is wired in.
func TestRenderBoard_AnsweredCollapsedByDefault(t *testing.T) {
	threads := []Thread{
		{ID: "o", From: "api", To: "web", Subject: "open one", Status: StatusOpen},
		{ID: "a", From: "api", To: "web", Subject: "answered one", Status: StatusAnswered},
	}
	var b strings.Builder
	if err := RenderBoard(&b, threads, "mission"); err != nil {
		t.Fatalf("RenderBoard: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, `data-thread="o" open>`) {
		t.Error("open thread should render expanded (open attribute)")
	}
	if !strings.Contains(out, `data-thread="a">`) {
		t.Error("answered thread should render collapsed (no open attribute)")
	}
	if strings.Contains(out, `data-thread="a" open>`) {
		t.Error("answered thread must not be expanded by default")
	}
	if !strings.Contains(out, "handoff:open:") {
		t.Error("board should include the collapse-state persistence script")
	}
}

// TestRenderBoard_CloseButtonOnEveryThread verifies the ✕ close control is
// rendered on every card regardless of status, so any thread can be closed.
func TestRenderBoard_CloseButtonOnEveryThread(t *testing.T) {
	mixed := []Thread{
		{ID: "o", From: "api", To: "web", Subject: "open one", Status: StatusOpen},
		{ID: "a", From: "api", To: "web", Subject: "answered one", Status: StatusAnswered},
	}
	var b strings.Builder
	if err := RenderBoard(&b, mixed, "mission"); err != nil {
		t.Fatalf("RenderBoard: %v", err)
	}
	if got := strings.Count(b.String(), `class="close-btn"`); got != 2 {
		t.Errorf("close-btn count = %d on a mixed board, want 2 (one per thread)", got)
	}

	var openOnly strings.Builder
	if err := RenderBoard(&openOnly, []Thread{{ID: "o", Subject: "open", Status: StatusOpen}}, "mission"); err != nil {
		t.Fatalf("RenderBoard: %v", err)
	}
	if !strings.Contains(openOnly.String(), `class="close-btn"`) {
		t.Error("an open-only board should still render a close button")
	}
}

// TestRenderBoard_TerminalCloseButton verifies the terminal variant also
// renders a close control on every thread.
func TestRenderBoard_TerminalCloseButton(t *testing.T) {
	mixed := []Thread{
		{ID: "o", From: "api", To: "web", Subject: "open one", Status: StatusOpen},
		{ID: "a", From: "api", To: "web", Subject: "answered one", Status: StatusAnswered},
	}
	var b strings.Builder
	if err := RenderBoard(&b, mixed, "terminal"); err != nil {
		t.Fatalf("RenderBoard: %v", err)
	}
	if got := strings.Count(b.String(), `class="thr-close"`); got != 2 {
		t.Errorf("thr-close count = %d on a mixed terminal board, want 2 (one per thread)", got)
	}
}

func TestRenderBoard_NewestFirst(t *testing.T) {
	threads := []Thread{
		{ID: "old", Subject: "older"},
		{ID: "new", Subject: "newer"},
	}
	var b strings.Builder
	if err := RenderBoard(&b, threads, "mission"); err != nil {
		t.Fatalf("RenderBoard error: %v", err)
	}
	out := b.String()
	if strings.Index(out, "newer") > strings.Index(out, "older") {
		t.Error("newest thread should appear before older thread")
	}
}

// TestRenderBoard_VariantsAndSwitcher checks every known variant renders, that
// an unknown/empty variant falls back to mission, and that all of them expose
// the style switcher with links to each variant.
func TestRenderBoard_VariantsAndSwitcher(t *testing.T) {
	for _, v := range []string{"mission", "terminal", "", "bogus"} {
		var b strings.Builder
		if err := RenderBoard(&b, nil, v); err != nil {
			t.Fatalf("RenderBoard(%q) error: %v", v, err)
		}
		out := b.String()
		for _, href := range []string{`href="?v=mission"`, `href="?v=terminal"`} {
			if !strings.Contains(out, href) {
				t.Errorf("variant %q missing switcher link %s", v, href)
			}
		}
		if strings.Contains(out, `href="?v=chat"`) {
			t.Errorf("variant %q should not expose the removed chat switcher link", v)
		}
	}
}

// TestRenderBoard_DefaultsToMission verifies an empty/unknown variant falls
// back to the mission-control board, not terminal.
func TestRenderBoard_DefaultsToMission(t *testing.T) {
	for _, v := range []string{"", "bogus"} {
		var b strings.Builder
		if err := RenderBoard(&b, nil, v); err != nil {
			t.Fatalf("RenderBoard(%q): %v", v, err)
		}
		if !strings.Contains(b.String(), "AGENT HANDOFF") {
			t.Errorf("variant %q should default to the mission-control board", v)
		}
	}
}

func TestRenderBoard_TerminalIsDistinct(t *testing.T) {
	threads := []Thread{{
		ID: "1", From: "api", To: "web", Subject: "S", Status: StatusOpen,
		Messages: []Message{{From: "api", Body: "hi", At: "2026-06-17T10:00:00Z"}},
	}}

	var mission, term strings.Builder
	if err := RenderBoard(&mission, threads, "mission"); err != nil {
		t.Fatalf("mission render: %v", err)
	}
	if err := RenderBoard(&term, threads, "terminal"); err != nil {
		t.Fatalf("terminal render: %v", err)
	}
	if !strings.Contains(term.String(), "handoff --watch") {
		t.Error("terminal variant should render the shell prompt")
	}
	if !strings.Contains(mission.String(), "AGENT HANDOFF") {
		t.Error("mission variant should render its header")
	}
}

// TestRenderBoard_ClipsLongMessages verifies the overview truncates long bodies
// and links out to the full-thread detail page rather than dumping everything.
func TestRenderBoard_ClipsLongMessages(t *testing.T) {
	long := strings.Repeat("lorem ipsum ", 200) // ~2400 chars
	threads := []Thread{{
		ID: "abc", From: "api", To: "web", Subject: "Long one", Status: StatusOpen,
		Messages: []Message{{From: "api", Body: long, At: "2026-06-17T10:00:00Z"}},
	}}
	for _, v := range []string{"mission", "terminal"} {
		var b strings.Builder
		if err := RenderBoard(&b, threads, v); err != nil {
			t.Fatalf("RenderBoard(%q): %v", v, err)
		}
		out := b.String()
		if !strings.Contains(out, "…") {
			t.Errorf("variant %q should clip long bodies with an ellipsis", v)
		}
		if strings.Contains(out, long) {
			t.Errorf("variant %q should not render the full long body on the overview", v)
		}
		if !strings.Contains(out, `href="/handoff/thread/abc?v=`+v+`"`) {
			t.Errorf("variant %q should link to the detail page preserving the variant", v)
		}
	}
}

// TestRenderBoard_ShowsOnlyRecentMessages verifies that a long thread only
// previews the newest few messages and reports how many are hidden.
func TestRenderBoard_ShowsOnlyRecentMessages(t *testing.T) {
	var msgs []Message
	for i := 0; i < 6; i++ {
		msgs = append(msgs, Message{From: "api", Body: "m" + string(rune('0'+i)), At: "2026-06-17T10:00:00Z"})
	}
	threads := []Thread{{ID: "1", From: "api", To: "web", Subject: "S", Status: StatusOpen, Messages: msgs}}

	var b strings.Builder
	if err := RenderBoard(&b, threads, "mission"); err != nil {
		t.Fatalf("RenderBoard: %v", err)
	}
	out := b.String()
	// 6 messages, previewCount=3 -> 3 hidden.
	if !strings.Contains(out, "+3 earlier message") {
		t.Errorf("overview should report 3 hidden messages, got:\n%s", out)
	}
	if strings.Contains(out, ">m0<") || strings.Contains(out, ">m2<") {
		t.Error("overview should not render the oldest messages")
	}
	if !strings.Contains(out, ">m5<") {
		t.Error("overview should render the newest message")
	}
}

// TestRenderThread_ShowsFullBodyAndEscapes verifies the detail page renders the
// complete (unclipped) conversation and still escapes HTML.
func TestRenderThread_ShowsFullBodyAndEscapes(t *testing.T) {
	long := strings.Repeat("data ", 300) // ~1500 chars, well over the clip
	thread := Thread{
		ID: "xyz", From: "api", To: "web", Subject: "Deep <dive>", Status: StatusAnswered,
		Messages: []Message{
			{From: "api", Body: long, At: "2026-06-17T10:00:00Z"},
			{From: "web", Body: "reply & <ok>", At: "2026-06-17T10:05:00Z"},
		},
	}
	for _, v := range []string{"mission", "terminal"} {
		var b strings.Builder
		if err := RenderThread(&b, thread, v); err != nil {
			t.Fatalf("RenderThread(%q): %v", v, err)
		}
		out := b.String()
		if !strings.Contains(out, long) {
			t.Errorf("variant %q detail page should render the full body", v)
		}
		if strings.Contains(out, "…") {
			t.Errorf("variant %q detail page should not clip messages", v)
		}
		if strings.Contains(out, "<dive>") || !strings.Contains(out, "Deep &lt;dive&gt;") {
			t.Errorf("variant %q detail page must escape the subject", v)
		}
		if !strings.Contains(out, "reply &amp; &lt;ok&gt;") {
			t.Errorf("variant %q detail page must escape message bodies", v)
		}
		if !strings.Contains(out, `href="/handoff?v=`+v+`"`) {
			t.Errorf("variant %q detail page should link back to the board", v)
		}
	}
}
