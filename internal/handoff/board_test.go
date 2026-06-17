package handoff

import (
	"strings"
	"testing"
)

func TestRenderBoard_Empty(t *testing.T) {
	var b strings.Builder
	if err := RenderBoard(&b, nil); err != nil {
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
	if err := RenderBoard(&b, threads); err != nil {
		t.Fatalf("RenderBoard error: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "api") || !strings.Contains(out, "web") {
		t.Error("board should show from and to")
	}
	if !strings.Contains(out, `status open`) {
		t.Error("board should color-code status via class")
	}
	if strings.Contains(out, "<script>") {
		t.Error("subject must be HTML-escaped, found raw <script>")
	}
	if !strings.Contains(out, "Hi &lt;script&gt;") {
		t.Error("subject should appear escaped")
	}
}

func TestRenderBoard_NewestFirst(t *testing.T) {
	threads := []Thread{
		{ID: "old", Subject: "older"},
		{ID: "new", Subject: "newer"},
	}
	var b strings.Builder
	if err := RenderBoard(&b, threads); err != nil {
		t.Fatalf("RenderBoard error: %v", err)
	}
	out := b.String()
	if strings.Index(out, "newer") > strings.Index(out, "older") {
		t.Error("newest thread should appear before older thread")
	}
}
