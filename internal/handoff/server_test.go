package handoff

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "handoff.json"))
	srv := httptest.NewServer(NewServer(store))
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestServer_FullLoop(t *testing.T) {
	srv := newTestServer(t)

	// create
	resp := postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"DB","body":"PK?"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("create Content-Type = %q, want json", ct)
	}
	var created Thread
	decode(t, resp, &created)
	if created.Status != StatusOpen || created.ID == "" {
		t.Fatalf("created thread wrong: %+v", created)
	}

	// list
	resp, _ = http.Get(srv.URL + "/handoff/threads")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	var list []Thread
	decode(t, resp, &list)
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// get
	resp, _ = http.Get(srv.URL + "/handoff/threads/" + created.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// answer -> status flips to answered
	resp = postJSON(t, srv.URL+"/handoff/threads/"+created.ID+"/messages", `{"from":"web","body":"bigint id"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("message status = %d, want 200", resp.StatusCode)
	}
	var answered Thread
	decode(t, resp, &answered)
	if answered.Status != StatusAnswered {
		t.Errorf("status after answer = %q, want answered", answered.Status)
	}

	// delete
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/handoff/threads/"+created.ID, nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", resp.StatusCode)
	}
	var del map[string]bool
	decode(t, resp, &del)
	if !del["deleted"] {
		t.Error("delete response should be {\"deleted\":true}")
	}
}

func TestServer_Close(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"closeme","body":"b"}`)
	var created Thread
	decode(t, resp, &created)

	resp = postJSON(t, srv.URL+"/handoff/threads/"+created.ID+"/close", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("close status = %d, want 200", resp.StatusCode)
	}
	var closed Thread
	decode(t, resp, &closed)
	if closed.Status != StatusClosed {
		t.Errorf("status after close = %q, want closed", closed.Status)
	}

	resp = postJSON(t, srv.URL+"/handoff/threads/missing/close", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("close missing status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_BoardHidesClosed(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"Hide me when closed","body":"b"}`)
	var created Thread
	decode(t, resp, &created)

	board, _ := http.Get(srv.URL + "/handoff")
	body, _ := io.ReadAll(board.Body)
	board.Body.Close()
	if !bytes.Contains(body, []byte("Hide me when closed")) {
		t.Fatal("board should show the thread before it is closed")
	}

	postJSON(t, srv.URL+"/handoff/threads/"+created.ID+"/close", "").Body.Close()

	board, _ = http.Get(srv.URL + "/handoff")
	body, _ = io.ReadAll(board.Body)
	board.Body.Close()
	if bytes.Contains(body, []byte("Hide me when closed")) {
		t.Error("board should hide closed threads")
	}

	// The API can still reach it explicitly.
	resp, _ = http.Get(srv.URL + "/handoff/threads?status=closed")
	var list []Thread
	decode(t, resp, &list)
	if len(list) != 1 {
		t.Errorf("status=closed returned %d, want 1", len(list))
	}
}

func TestServer_404Shape(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := http.Get(srv.URL + "/handoff/threads/missing")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var body map[string]string
	decode(t, resp, &body)
	if body["message"] != "Thread not found." {
		t.Errorf("404 message = %q, want %q", body["message"], "Thread not found.")
	}
}

func TestServer_ValidationErrors(t *testing.T) {
	srv := newTestServer(t)
	tests := []struct {
		name string
		body string
		bad  string // field expected in errors
	}{
		{"missing all", `{}`, "from"},
		{"bad from name", `{"from":"Bad_Name","to":"web","subject":"s","body":"b"}`, "from"},
		{"over-length subject", `{"from":"api","to":"web","subject":"` + strings.Repeat("x", 201) + `","body":"b"}`, "subject"},
		{"missing body", `{"from":"api","to":"web","subject":"s"}`, "body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := postJSON(t, srv.URL+"/handoff/threads", tt.body)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", resp.StatusCode)
			}
			var body struct {
				Message string              `json:"message"`
				Errors  map[string][]string `json:"errors"`
			}
			decode(t, resp, &body)
			if body.Message != "Validation failed." {
				t.Errorf("message = %q, want %q", body.Message, "Validation failed.")
			}
			if len(body.Errors[tt.bad]) == 0 {
				t.Errorf("expected errors for field %q, got %+v", tt.bad, body.Errors)
			}
		})
	}
}

func TestServer_Filters(t *testing.T) {
	srv := newTestServer(t)
	postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"s1","body":"b"}`).Body.Close()
	postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"db","subject":"s2","body":"b"}`).Body.Close()

	resp, _ := http.Get(srv.URL + "/handoff/threads?to=db")
	var list []Thread
	decode(t, resp, &list)
	if len(list) != 1 || list[0].To != "db" {
		t.Errorf("filter to=db returned %d threads", len(list))
	}
}

func TestServer_AlwaysJSON(t *testing.T) {
	srv := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/handoff/threads", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /handoff/threads: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want json even with Accept: text/html", ct)
	}
}

func TestServer_BoardIsHTML(t *testing.T) {
	srv := newTestServer(t)
	postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"Board me","body":"b"}`).Body.Close()
	resp, err := http.Get(srv.URL + "/handoff")
	if err != nil {
		t.Fatalf("GET /handoff: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("board status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("board Content-Type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("Board me")) {
		t.Error("board should render the thread subject")
	}
}

func TestServer_BoardVariant(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/handoff?v=terminal")
	if err != nil {
		t.Fatalf("GET /handoff?v=terminal: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "handoff --watch") {
		t.Error("?v=terminal should render the terminal variant")
	}
}

func TestServer_ThreadPageIsHTML(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"Detail me","body":"the full body"}`)
	var created Thread
	decode(t, resp, &created)

	resp, err := http.Get(srv.URL + "/handoff/thread/" + created.ID + "?v=terminal")
	if err != nil {
		t.Fatalf("GET thread page: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("thread page status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("thread page Content-Type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("the full body")) {
		t.Error("thread page should render the full message body")
	}
	if !bytes.Contains(body, []byte("handoff --thread")) {
		t.Error("?v=terminal thread page should render the terminal variant")
	}
}

func TestServer_ThreadPageNotFound(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/handoff/thread/missing")
	if err != nil {
		t.Fatalf("GET missing thread page: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestServer_GetThreadShape(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"shape","body":"q"}`)
	var created Thread
	decode(t, resp, &created)

	resp, _ = http.Get(srv.URL + "/handoff/threads/" + created.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", resp.StatusCode)
	}
	var got Thread
	decode(t, resp, &got)
	if got.ID != created.ID || got.From != "api" || got.To != "web" || got.Subject != "shape" {
		t.Errorf("got thread = %+v, want fields to match created", got)
	}
	if got.Status != StatusOpen {
		t.Errorf("got.Status = %q, want %q", got.Status, StatusOpen)
	}
	if len(got.Messages) != 1 || got.Messages[0].Body != "q" {
		t.Errorf("got.Messages = %+v, want 1 message with body q", got.Messages)
	}
}

func TestServer_MalformedJSONReturns400(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/handoff/threads", `{"from":`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed body status = %d, want 400", resp.StatusCode)
	}
}

func TestServer_TrailingJSONReturns400(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"s","body":"b"}{"x":1}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("trailing-data body status = %d, want 400", resp.StatusCode)
	}
}

func TestServer_BoardRefreshNotLogged(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	store := NewStore(filepath.Join(t.TempDir(), "handoff.json"))
	h := NewServer(store)

	// The board auto-refreshes constantly, so GET /handoff must not be logged.
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/handoff", nil))
	if strings.Contains(buf.String(), "GET /handoff 200") {
		t.Errorf("board GET should not be logged, got: %q", buf.String())
	}

	// API requests are still logged.
	buf.Reset()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/handoff/threads", nil))
	if !strings.Contains(buf.String(), "GET /handoff/threads 200") {
		t.Errorf("API GET should be logged, got: %q", buf.String())
	}
}

func TestServer_LongPollWaitsForThread(t *testing.T) {
	srv := newTestServer(t)
	go func() {
		time.Sleep(30 * time.Millisecond)
		postJSON(t, srv.URL+"/handoff/threads", `{"from":"web","to":"api","subject":"s","body":"q"}`).Body.Close()
	}()
	resp, err := http.Get(srv.URL + "/handoff/threads?to=api&status=open&wait=2")
	if err != nil {
		t.Fatalf("long-poll GET: %v", err)
	}
	var list []Thread
	decode(t, resp, &list)
	if len(list) != 1 {
		t.Fatalf("long poll should return the thread once it is posted, got %d", len(list))
	}
}

func TestServer_LongPollTimesOutEmpty(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/handoff/threads?to=api&status=open&wait=500ms")
	if err != nil {
		t.Fatalf("long-poll GET: %v", err)
	}
	var list []Thread
	decode(t, resp, &list)
	if len(list) != 0 {
		t.Errorf("long poll should return an empty list on timeout, got %d", len(list))
	}
}

func TestServer_StatusFilter(t *testing.T) {
	srv := newTestServer(t)
	postJSON(t, srv.URL+"/handoff/threads", `{"from":"api","to":"web","subject":"s","body":"b"}`).Body.Close()

	resp, _ := http.Get(srv.URL + "/handoff/threads?status=open")
	var open []Thread
	decode(t, resp, &open)
	if len(open) != 1 {
		t.Errorf("status=open returned %d, want 1", len(open))
	}

	resp, _ = http.Get(srv.URL + "/handoff/threads?status=answered")
	var answered []Thread
	decode(t, resp, &answered)
	if len(answered) != 0 {
		t.Errorf("status=answered returned %d, want 0", len(answered))
	}
}
