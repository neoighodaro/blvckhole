package handoff

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "handoff.json"))
}

func TestStore_MissingFileIsEmpty(t *testing.T) {
	s := newTestStore(t)
	threads, err := s.All("")
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("All() len = %d, want 0", len(threads))
	}
}

func TestStore_MalformedFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "handoff.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(path)
	threads, err := s.All("")
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	if len(threads) != 0 {
		t.Errorf("All() len = %d, want 0 for malformed file", len(threads))
	}
}

func TestStore_CorruptFileIsPreservedNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "handoff.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(path)

	// A mutating call must not silently destroy the unparseable contents.
	if _, err := s.Open("api", "web", "s", "b"); err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	backup, err := os.ReadFile(path + ".corrupt")
	if err != nil {
		t.Fatalf("corrupt backup not preserved: %v", err)
	}
	if string(backup) != "{not json" {
		t.Errorf("corrupt backup = %q, want original bytes", backup)
	}

	// The store recovers and persists new data going forward.
	threads, err := s.All("")
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	if len(threads) != 1 {
		t.Errorf("All() len = %d, want 1 after recovery", len(threads))
	}
}

func TestStore_OpenCreatesThread(t *testing.T) {
	s := newTestStore(t)
	th, err := s.Open("api", "web", "DB schema", "What is the PK?")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if th.ID == "" {
		t.Error("Open() thread has empty ID")
	}
	if th.Status != StatusOpen {
		t.Errorf("status = %q, want %q", th.Status, StatusOpen)
	}
	if th.From != "api" || th.To != "web" || th.Subject != "DB schema" {
		t.Errorf("thread fields wrong: %+v", th)
	}
	if len(th.Messages) != 1 || th.Messages[0].From != "api" || th.Messages[0].Body != "What is the PK?" {
		t.Errorf("messages wrong: %+v", th.Messages)
	}
	if th.CreatedAt == "" || th.UpdatedAt == "" {
		t.Error("timestamps not set")
	}
}

func TestStore_FindReturnsNilWhenMissing(t *testing.T) {
	s := newTestStore(t)
	th, err := s.Find("nope")
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}
	if th != nil {
		t.Errorf("Find() = %+v, want nil", th)
	}
}

func TestStore_FindReturnsThread(t *testing.T) {
	s := newTestStore(t)
	created, _ := s.Open("api", "web", "s", "b")
	found, err := s.Find(created.ID)
	if err != nil {
		t.Fatalf("Find() error: %v", err)
	}
	if found == nil || found.ID != created.ID {
		t.Errorf("Find() = %+v, want id %q", found, created.ID)
	}
}

func TestStore_AddMessageExplicitIntent(t *testing.T) {
	s := newTestStore(t)
	th, _ := s.Open("api", "web", "s", "q1") // open, waiting_on=web

	// Recipient replies handing the ball back to the asker (clarify).
	th, err := s.AddMessage(th.ID, "web", "need detail?", "api", false)
	if err != nil {
		t.Fatalf("AddMessage error: %v", err)
	}
	if th.Status != StatusOpen || th.WaitingOn != "api" {
		t.Errorf("after clarify reply: status=%q waiting_on=%q, want open/api", th.Status, th.WaitingOn)
	}
	if len(th.Messages) != 2 {
		t.Errorf("messages len = %d, want 2", len(th.Messages))
	}

	// Asker replies and resolves the thread.
	th, _ = s.AddMessage(th.ID, "api", "here it is", "", true)
	if th.Status != StatusAnswered || th.WaitingOn != "" {
		t.Errorf("after resolve: status=%q waiting_on=%q, want answered/\"\"", th.Status, th.WaitingOn)
	}
}

func TestStore_AddMessageMissingThread(t *testing.T) {
	s := newTestStore(t)
	th, err := s.AddMessage("nope", "web", "a", "api", false)
	if err != nil {
		t.Fatalf("AddMessage error: %v", err)
	}
	if th != nil {
		t.Errorf("AddMessage() = %+v, want nil for missing thread", th)
	}
}

func TestStore_Delete(t *testing.T) {
	s := newTestStore(t)
	th, _ := s.Open("api", "web", "s", "b")
	ok, err := s.Delete(th.ID)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if !ok {
		t.Error("Delete() = false, want true")
	}
	again, _ := s.Delete(th.ID)
	if again {
		t.Error("second Delete() = true, want false")
	}
}

func TestStore_Close(t *testing.T) {
	s := newTestStore(t)
	th, _ := s.Open("api", "web", "s", "b")
	s.AddMessage(th.ID, "web", "answer", "", true) // -> answered

	closed, err := s.Close(th.ID)
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
	if closed == nil || closed.Status != StatusClosed {
		t.Fatalf("Close() = %+v, want status closed", closed)
	}

	found, _ := s.Find(th.ID)
	if found == nil || found.Status != StatusClosed {
		t.Errorf("after Close, persisted status = %+v, want closed", found)
	}

	missing, err := s.Close("nope")
	if err != nil {
		t.Fatalf("Close(missing) error: %v", err)
	}
	if missing != nil {
		t.Errorf("Close(missing) = %+v, want nil", missing)
	}
}

func TestStore_CloseClearsWaitingOn(t *testing.T) {
	s := newTestStore(t)
	th, _ := s.Open("api", "web", "s", "q") // open, waiting_on=web

	closed, err := s.Close(th.ID)
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
	if closed.Status != StatusClosed || closed.WaitingOn != "" {
		t.Errorf("after closing an open thread: status=%q waiting_on=%q, want closed/\"\"", closed.Status, closed.WaitingOn)
	}

	// A closed thread is nobody's turn, so the waiting_on watch must not surface
	// it (otherwise a long-poll on the old recipient would spin).
	web, _ := s.All("web")
	if len(web) != 0 {
		t.Errorf("All(waiting_on=web) returned %d threads, want 0 after close", len(web))
	}
}

func TestStore_MessageReopensClosed(t *testing.T) {
	s := newTestStore(t)
	th, _ := s.Open("api", "web", "s", "b")
	if _, err := s.Close(th.ID); err != nil {
		t.Fatalf("Close error: %v", err)
	}
	// Closing is not permanent: an explicit-intent reply reopens it.
	reopened, _ := s.AddMessage(th.ID, "web", "actually, an update", "api", false)
	if reopened.Status != StatusOpen || reopened.WaitingOn != "api" {
		t.Errorf("after reply on closed thread: status=%q waiting_on=%q, want open/api", reopened.Status, reopened.WaitingOn)
	}
}

func TestStore_AllFiltersByWaitingOn(t *testing.T) {
	s := newTestStore(t)
	s.Open("api", "web", "s1", "b") // open, waiting_on=web
	s.Open("api", "db", "s2", "b")  // open, waiting_on=db

	web, _ := s.All("web")
	if len(web) != 1 || web[0].To != "web" {
		t.Errorf("All(waiting_on=web) = %d threads, want 1", len(web))
	}
	db, _ := s.All("db")
	if len(db) != 1 || db[0].To != "db" {
		t.Errorf("All(waiting_on=db) = %d threads, want 1", len(db))
	}
	all, _ := s.All("")
	if len(all) != 2 {
		t.Errorf("All(\"\") = %d threads, want 2 (no filter returns all)", len(all))
	}
}

func TestStore_ConcurrentOpensNoLostWrites(t *testing.T) {
	s := newTestStore(t)
	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Open("api", "web", "s", "b"); err != nil {
				t.Errorf("concurrent Open error: %v", err)
			}
		}()
	}
	wg.Wait()
	threads, _ := s.All("")
	if len(threads) != n {
		t.Errorf("after %d concurrent opens, All() len = %d", n, len(threads))
	}
}

func TestStore_WaitForReturnsImmediatelyWhenMatchExists(t *testing.T) {
	s := newTestStore(t)
	s.Open("web", "api", "s", "q") // open thread addressed to api
	got, err := s.WaitFor(context.Background(), "api", 2*time.Second)
	if err != nil {
		t.Fatalf("WaitFor: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("WaitFor returned %d threads, want 1 immediately", len(got))
	}
}

func TestStore_WaitForBlocksUntilThreadAppears(t *testing.T) {
	s := newTestStore(t)
	go func() {
		time.Sleep(20 * time.Millisecond)
		s.Open("web", "api", "s", "q")
	}()
	got, err := s.WaitFor(context.Background(), "api", 2*time.Second)
	if err != nil {
		t.Fatalf("WaitFor: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("WaitFor returned %d threads, want 1 after the thread is opened mid-wait", len(got))
	}
}

func TestStore_WaitForWakesOnFollowUp(t *testing.T) {
	s := newTestStore(t)
	th, _ := s.Open("web", "api", "s", "q")        // to=api, open
	s.AddMessage(th.ID, "api", "answer", "", true) // api resolves -> waiting_on cleared
	go func() {
		time.Sleep(20 * time.Millisecond)
		s.AddMessage(th.ID, "web", "follow up", "api", false) // asker follows up -> waiting_on=api
	}()
	got, err := s.WaitFor(context.Background(), "api", 2*time.Second)
	if err != nil {
		t.Fatalf("WaitFor: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("a follow-up should reopen the thread and wake the waiter, got %d", len(got))
	}
}

func TestStore_WaitForTimesOutEmpty(t *testing.T) {
	s := newTestStore(t)
	got, err := s.WaitFor(context.Background(), "api", 30*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitFor: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("WaitFor should return empty on timeout, got %d", len(got))
	}
}

func TestStore_WaitForZeroIsImmediate(t *testing.T) {
	s := newTestStore(t)
	got, err := s.WaitFor(context.Background(), "api", 0)
	if err != nil {
		t.Fatalf("WaitFor: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("WaitFor(0) should behave like All and return immediately, got %d", len(got))
	}
}

func TestDefaultPidAndLogPath_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if got, want := DefaultPidPath(), filepath.Join("/tmp/xdg", "blvckhole", "handoff", "handoff.pid"); got != want {
		t.Errorf("DefaultPidPath() = %q, want %q", got, want)
	}
	if got, want := DefaultLogPath(), filepath.Join("/tmp/xdg", "blvckhole", "handoff", "handoff.log"); got != want {
		t.Errorf("DefaultLogPath() = %q, want %q", got, want)
	}
}

func TestDefaultStorePath_XDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	got := DefaultStorePath()
	want := filepath.Join("/tmp/xdg", "blvckhole", "handoff", "handoff.json")
	if got != want {
		t.Errorf("DefaultStorePath() = %q, want %q", got, want)
	}
}

func TestDefaultStorePath_HomeFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/fakehome")
	got := DefaultStorePath()
	want := filepath.Join("/tmp/fakehome", ".config", "blvckhole", "handoff", "handoff.json")
	if got != want {
		t.Errorf("DefaultStorePath() = %q, want %q", got, want)
	}
}

func TestStore_OpenSetsWaitingOnRecipient(t *testing.T) {
	s := newTestStore(t)
	th, err := s.Open("api", "web", "s", "q")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if th.WaitingOn != "web" {
		t.Errorf("WaitingOn = %q, want %q (the recipient)", th.WaitingOn, "web")
	}
}

func TestStore_LoadBackfillsWaitingOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "handoff.json")
	legacy := `{"threads":[
      {"id":"leg-open","from":"api","to":"web","subject":"s","status":"open","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","messages":[{"from":"api","body":"q","at":"2026-01-01T00:00:00Z"}]},
      {"id":"leg-open-recip","from":"api","to":"web","subject":"s","status":"open","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","messages":[{"from":"api","body":"q","at":"2026-01-01T00:00:00Z"},{"from":"web","body":"clarify?","at":"2026-01-01T00:01:00Z"}]},
      {"id":"leg-ans","from":"api","to":"web","subject":"s","status":"answered","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","messages":[{"from":"api","body":"q","at":"2026-01-01T00:00:00Z"},{"from":"web","body":"a","at":"2026-01-01T00:01:00Z"}]},
      {"id":"leg-closed","from":"api","to":"web","subject":"s","status":"closed","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","messages":[{"from":"api","body":"q","at":"2026-01-01T00:00:00Z"}]}
    ]}`
	if err := os.WriteFile(path, []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(path)

	cases := map[string]string{
		"leg-open":       "web", // last msg from asker -> recipient's turn
		"leg-open-recip": "api", // last msg from recipient -> asker's turn
		"leg-ans":        "",    // resolved -> nobody's turn
		"leg-closed":     "",    // closed -> nobody's turn
	}
	for id, want := range cases {
		th, err := s.Find(id)
		if err != nil || th == nil {
			t.Fatalf("Find(%q) = %+v, err %v", id, th, err)
		}
		if th.WaitingOn != want {
			t.Errorf("thread %q WaitingOn = %q, want %q", id, th.WaitingOn, want)
		}
	}
}
