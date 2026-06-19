package handoff

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	StatusOpen     = "open"
	StatusAnswered = "answered"
	StatusClosed   = "closed"
)

type Message struct {
	From string `json:"from"`
	Body string `json:"body"`
	At   string `json:"at"`
}

type Thread struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"`
	WaitingOn string    `json:"waiting_on"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	Messages  []Message `json:"messages"`
}

type fileData struct {
	Threads []Thread `json:"threads"`
}

// Store is a JSON-file-backed thread store. A single mutex serializes every
// read-modify-write; there is exactly one broker process so no file lock is
// needed.
type Store struct {
	path string
	mu   sync.Mutex

	// notifyCh is closed (and replaced) every time the store changes, so
	// long-poll waiters can block until something happens. It is guarded by its
	// own mutex to keep it independent of the data lock.
	notifyMu sync.Mutex
	notifyCh chan struct{}
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

// subscribe returns a channel that is closed the next time the store changes.
// Callers must grab it before reading state so a concurrent mutation cannot slip
// in between the read and the wait.
func (s *Store) subscribe() <-chan struct{} {
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	if s.notifyCh == nil {
		s.notifyCh = make(chan struct{})
	}
	return s.notifyCh
}

// broadcast wakes every current subscriber.
func (s *Store) broadcast() {
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	if s.notifyCh != nil {
		close(s.notifyCh)
		s.notifyCh = nil
	}
}

// handoffDir resolves the shared host config directory for the broker.
func handoffDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "blvckhole", "handoff")
}

// DefaultStorePath resolves the shared host store path.
func DefaultStorePath() string { return filepath.Join(handoffDir(), "handoff.json") }

// DefaultPidPath is where a backgrounded broker records its PID.
func DefaultPidPath() string { return filepath.Join(handoffDir(), "handoff.pid") }

// DefaultLogPath is where a backgrounded broker writes its output.
func DefaultLogPath() string { return filepath.Join(handoffDir(), "handoff.log") }

// load reads the store, tolerating a missing or malformed file as empty.
func (s *Store) load() ([]Thread, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Thread{}, nil
		}
		return nil, err
	}
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		// The file exists but is unparseable. Preserve it (best-effort) so the
		// next save() does not silently overwrite recoverable data, then treat
		// the store as empty.
		_ = os.Rename(s.path, s.path+".corrupt")
		return []Thread{}, nil
	}
	if fd.Threads == nil {
		return []Thread{}, nil
	}
	threads := fd.Threads
	for i := range threads {
		if threads[i].WaitingOn != "" {
			continue
		}
		// Legacy threads predate waiting_on. Only an open thread has an
		// outstanding turn; answered/closed are resolved (nobody's turn).
		if threads[i].Status == StatusOpen {
			threads[i].WaitingOn = turnFromLastMessage(threads[i])
		}
	}
	return threads, nil
}

// turnFromLastMessage infers whose turn an open legacy thread is on from the
// author of its most recent message: a message from the asker hands the ball to
// the recipient; anything else hands it back to the asker.
func turnFromLastMessage(t Thread) string {
	if len(t.Messages) == 0 {
		return t.To
	}
	if t.Messages[len(t.Messages)-1].From == t.From {
		return t.To
	}
	return t.From
}

// save writes atomically: tmp file then rename.
func (s *Store) save(threads []Thread) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(fileData{Threads: threads}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	// fsync the temp file before the rename so a crash cannot leave the renamed
	// store pointing at unflushed (possibly garbage) contents.
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// All returns every thread, or only those whose turn is filterWaitingOn when it
// is non-empty.
func (s *Store) All(filterWaitingOn string) ([]Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threads, err := s.load()
	if err != nil {
		return nil, err
	}
	result := make([]Thread, 0, len(threads))
	for _, t := range threads {
		if filterWaitingOn != "" && t.WaitingOn != filterWaitingOn {
			continue
		}
		result = append(result, t)
	}
	return result, nil
}

// WaitFor is a long-poll All: with wait <= 0 it returns immediately, otherwise it
// blocks up to wait for at least one matching thread to appear. It re-queries on
// every store change, so a thread whose turn becomes filterWaitingOn mid-wait
// returns right away; on timeout (or a cancelled ctx) it returns the current —
// empty — result.
func (s *Store) WaitFor(ctx context.Context, filterWaitingOn string, wait time.Duration) ([]Thread, error) {
	if wait <= 0 {
		return s.All(filterWaitingOn)
	}
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	for {
		ch := s.subscribe()
		threads, err := s.All(filterWaitingOn)
		if err != nil || len(threads) > 0 {
			return threads, err
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return threads, nil
		}
	}
}

func (s *Store) Find(id string) (*Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threads, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range threads {
		if threads[i].ID == id {
			t := threads[i]
			return &t, nil
		}
	}
	return nil, nil
}

func (s *Store) Open(from, to, subject, body string) (*Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threads, err := s.load()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	t := Thread{
		ID:        NewID(),
		From:      from,
		To:        to,
		Subject:   subject,
		Status:    StatusOpen,
		WaitingOn: to,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{{From: from, Body: body, At: now}},
	}
	threads = append(threads, t)
	if err := s.save(threads); err != nil {
		return nil, err
	}
	s.broadcast()
	return &t, nil
}

// AddMessage appends a message and applies the caller's explicit turn intent:
// markAnswered resolves the thread (status=answered, no waiting party); otherwise
// the thread stays open and waitingOn names whose turn it is next. Returns the
// updated thread, or nil if no thread has the id.
func (s *Store) AddMessage(id, from, body, waitingOn string, markAnswered bool) (*Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threads, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range threads {
		if threads[i].ID == id {
			now := time.Now().UTC().Format(time.RFC3339)
			threads[i].Messages = append(threads[i].Messages, Message{From: from, Body: body, At: now})
			if markAnswered {
				threads[i].Status = StatusAnswered
				threads[i].WaitingOn = ""
			} else {
				threads[i].Status = StatusOpen
				threads[i].WaitingOn = waitingOn
			}
			threads[i].UpdatedAt = now
			if err := s.save(threads); err != nil {
				return nil, err
			}
			s.broadcast()
			t := threads[i]
			return &t, nil
		}
	}
	return nil, nil
}

// Close marks a thread as manually closed so it drops off the board. It is not
// a permanent state: a later explicit-intent reply via AddMessage reopens the
// thread (a reply with waiting_on set), while an answered reply leaves it
// resolved. Returns the updated thread, or nil if no thread has the id.
func (s *Store) Close(id string) (*Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threads, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range threads {
		if threads[i].ID == id {
			threads[i].Status = StatusClosed
			// A closed thread is nobody's turn: clearing waiting_on keeps it off
			// the waiting_on watch (so a closed-while-open thread can't spin a
			// long-poll) and matches the answered/closed = "" data model.
			threads[i].WaitingOn = ""
			if err := s.save(threads); err != nil {
				return nil, err
			}
			s.broadcast()
			t := threads[i]
			return &t, nil
		}
	}
	return nil, nil
}

func (s *Store) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threads, err := s.load()
	if err != nil {
		return false, err
	}
	for i := range threads {
		if threads[i].ID == id {
			threads = append(threads[:i], threads[i+1:]...)
			if err := s.save(threads); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}
