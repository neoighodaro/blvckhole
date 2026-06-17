package handoff

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// maxWait caps how long a long-poll request may park a connection, so a client
// cannot hold one open forever.
const maxWait = 5 * time.Minute

type createReq struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type messageReq struct {
	From string `json:"from"`
	Body string `json:"body"`
}

// NewServer returns the handoff HTTP handler with request logging.
func NewServer(store *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /handoff", handleBoard(store))
	mux.HandleFunc("GET /handoff/thread/{id}", handleThreadPage(store))
	mux.HandleFunc("GET /handoff/threads", handleList(store))
	mux.HandleFunc("POST /handoff/threads", handleCreate(store))
	mux.HandleFunc("GET /handoff/threads/{id}", handleGet(store))
	mux.HandleFunc("POST /handoff/threads/{id}/messages", handleMessage(store))
	mux.HandleFunc("DELETE /handoff/threads/{id}", handleDelete(store))
	return withLogging(mux)
}

func handleBoard(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threads, err := store.All("", "")
		if err != nil {
			// The board is an HTML endpoint; don't answer it with a JSON body.
			http.Error(w, "Failed to load threads.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := RenderBoard(w, threads, r.URL.Query().Get("v")); err != nil {
			log.Printf("board render error: %v", err)
		}
	}
}

func handleThreadPage(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		thread, err := store.Find(r.PathValue("id"))
		if err != nil {
			// HTML endpoint; don't answer it with a JSON body.
			http.Error(w, "Failed to load thread.", http.StatusInternalServerError)
			return
		}
		if thread == nil {
			http.Error(w, "Thread not found.", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := RenderThread(w, *thread, r.URL.Query().Get("v")); err != nil {
			log.Printf("thread render error: %v", err)
		}
	}
}

func handleList(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		threads, err := store.WaitFor(r.Context(), q.Get("to"), q.Get("status"), parseWait(q.Get("wait")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to load threads.")
			return
		}
		writeJSON(w, http.StatusOK, threads)
	}
}

// parseWait reads the long-poll ?wait value. It accepts a bare number of seconds
// ("25") or a Go duration ("25s", "2m"). Anything invalid or non-positive means
// no waiting; the result is capped at maxWait.
func parseWait(v string) time.Duration {
	if v == "" {
		return 0
	}
	var d time.Duration
	if n, err := strconv.Atoi(v); err == nil {
		d = time.Duration(n) * time.Second
	} else if pd, err := time.ParseDuration(v); err == nil {
		d = pd
	} else {
		return 0
	}
	if d < 0 {
		return 0
	}
	if d > maxWait {
		d = maxWait
	}
	return d
}

func handleCreate(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createReq
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body.")
			return
		}
		errs := map[string][]string{}
		validateName(errs, "from", req.From)
		validateName(errs, "to", req.To)
		validateRequiredMax(errs, "subject", req.Subject, 200)
		if req.Body == "" {
			errs["body"] = append(errs["body"], "The body field is required.")
		}
		if len(errs) > 0 {
			writeValidationError(w, errs)
			return
		}
		thread, err := store.Open(req.From, req.To, req.Subject, req.Body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create thread.")
			return
		}
		writeJSON(w, http.StatusCreated, thread)
	}
}

func handleGet(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		thread, err := store.Find(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to load thread.")
			return
		}
		if thread == nil {
			writeError(w, http.StatusNotFound, "Thread not found.")
			return
		}
		writeJSON(w, http.StatusOK, thread)
	}
}

func handleMessage(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req messageReq
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body.")
			return
		}
		errs := map[string][]string{}
		validateName(errs, "from", req.From)
		if req.Body == "" {
			errs["body"] = append(errs["body"], "The body field is required.")
		}
		if len(errs) > 0 {
			writeValidationError(w, errs)
			return
		}
		thread, err := store.AddMessage(r.PathValue("id"), req.From, req.Body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to add message.")
			return
		}
		if thread == nil {
			writeError(w, http.StatusNotFound, "Thread not found.")
			return
		}
		writeJSON(w, http.StatusOK, thread)
	}
}

func handleDelete(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deleted, err := store.Delete(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to delete thread.")
			return
		}
		if !deleted {
			writeError(w, http.StatusNotFound, "Thread not found.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	}
}

// validateName enforces required, <=50, and the sandbox-name regex.
func validateName(errs map[string][]string, field, value string) {
	if value == "" {
		errs[field] = append(errs[field], "The "+field+" field is required.")
		return
	}
	if len(value) > 50 {
		errs[field] = append(errs[field], "The "+field+" may not be greater than 50 characters.")
	}
	if !nameRe.MatchString(value) {
		errs[field] = append(errs[field], "The "+field+" must be a valid sandbox name.")
	}
}

func validateRequiredMax(errs map[string][]string, field, value string, max int) {
	if value == "" {
		errs[field] = append(errs[field], "The "+field+" field is required.")
		return
	}
	if len(value) > max {
		errs[field] = append(errs[field], "The "+field+" is too long.")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil // empty body -> zero value -> validation handles it
		}
		return err
	}
	// Decode reads only the first JSON value; reject any trailing data so a
	// body like `{...}{...}` is a 400 rather than a silently-ignored remainder.
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return errors.New("unexpected trailing data after JSON body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func writeValidationError(w http.ResponseWriter, errs map[string][]string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"message": "Validation failed.",
		"errors":  errs,
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		// The board auto-refreshes every few seconds; logging each GET /handoff
		// floods the terminal, so skip it. API requests are still logged.
		if r.Method == http.MethodGet && r.URL.Path == "/handoff" {
			return
		}
		log.Printf("%s %s %d", r.Method, r.URL.Path, rec.status)
	})
}
