package handoff

import (
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
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

// DefaultStorePath resolves the shared host store path.
func DefaultStorePath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "blvckhole", "handoff", "handoff.json")
}

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
	return fd.Threads, nil
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

func (s *Store) All(filterTo, filterStatus string) ([]Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	threads, err := s.load()
	if err != nil {
		return nil, err
	}
	result := make([]Thread, 0, len(threads))
	for _, t := range threads {
		if filterTo != "" && t.To != filterTo {
			continue
		}
		if filterStatus != "" && t.Status != filterStatus {
			continue
		}
		result = append(result, t)
	}
	return result, nil
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
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{{From: from, Body: body, At: now}},
	}
	threads = append(threads, t)
	if err := s.save(threads); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) AddMessage(id, from, body string) (*Thread, error) {
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
			// Status is derived, never client-set: asker follow-up reopens,
			// anyone else answering closes.
			if from == threads[i].From {
				threads[i].Status = StatusOpen
			} else {
				threads[i].Status = StatusAnswered
			}
			threads[i].UpdatedAt = now
			if err := s.save(threads); err != nil {
				return nil, err
			}
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
