package handoff

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "handoff.json"))
}

func TestStore_MissingFileIsEmpty(t *testing.T) {
	s := newTestStore(t)
	threads, err := s.All("", "")
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
	threads, err := s.All("", "")
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
	threads, err := s.All("", "")
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

func TestStore_StatusFlipMatrix(t *testing.T) {
	s := newTestStore(t)
	th, _ := s.Open("api", "web", "s", "q1")
	if th.Status != StatusOpen {
		t.Fatalf("after open: status = %q, want open", th.Status)
	}

	// recipient answers -> answered
	th, err := s.AddMessage(th.ID, "web", "a1")
	if err != nil {
		t.Fatalf("AddMessage error: %v", err)
	}
	if th.Status != StatusAnswered {
		t.Errorf("after recipient reply: status = %q, want answered", th.Status)
	}
	if len(th.Messages) != 2 {
		t.Errorf("messages len = %d, want 2", len(th.Messages))
	}

	// original asker follows up -> open
	th, _ = s.AddMessage(th.ID, "api", "q2")
	if th.Status != StatusOpen {
		t.Errorf("after asker follow-up: status = %q, want open", th.Status)
	}

	// recipient answers again -> answered
	th, _ = s.AddMessage(th.ID, "web", "a2")
	if th.Status != StatusAnswered {
		t.Errorf("after second reply: status = %q, want answered", th.Status)
	}
}

func TestStore_AddMessageMissingThread(t *testing.T) {
	s := newTestStore(t)
	th, err := s.AddMessage("nope", "web", "a")
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

func TestStore_AllFilters(t *testing.T) {
	s := newTestStore(t)
	a, _ := s.Open("api", "web", "s1", "b") // open, to=web
	s.Open("api", "db", "s2", "b")          // open, to=db
	s.AddMessage(a.ID, "web", "answer")     // a -> answered, to=web

	byTo, _ := s.All("web", "")
	if len(byTo) != 1 || byTo[0].To != "web" {
		t.Errorf("All(to=web) = %d threads, want 1", len(byTo))
	}
	byStatus, _ := s.All("", StatusOpen)
	if len(byStatus) != 1 || byStatus[0].Status != StatusOpen {
		t.Errorf("All(status=open) = %d threads, want 1", len(byStatus))
	}
	both, _ := s.All("web", StatusAnswered)
	if len(both) != 1 {
		t.Errorf("All(to=web,status=answered) = %d, want 1", len(both))
	}
	none, _ := s.All("web", StatusOpen)
	if len(none) != 0 {
		t.Errorf("All(to=web,status=open) = %d, want 0", len(none))
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
	threads, _ := s.All("", "")
	if len(threads) != n {
		t.Errorf("after %d concurrent opens, All() len = %d", n, len(threads))
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
