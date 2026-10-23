# 06 — E2E Tests

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Write app-level e2e tests using `httptest` + in-memory SQLite + goose migrations, and svc-level unit tests for validation logic. A shared DB test helper sets up the temp DB.

**Architecture:** `internal/app/e2e_test.go` uses `httptest.NewServer` with the app `NewHandler` backed by a real SQLite in-memory DB. Tests exercise the full HTTP contract: create note, list notes, get note, delete note, create log entry, list log entries — including validation failures (400), not-found (404), pagination, filters, and payload round-trip. Svc-level unit tests isolate translation + validation logic with mock stores (already written in sub-plan 04). [S18].

**Tech Stack:** Go stdlib (`net/http`, `net/http/httptest`, `encoding/json`, `testing`), `port/state/sqlite`, `app`, `svc`, goose v3.

**Depends on:** All prior sub-plans (01: adapter interfaces, 02: SQLite ports, 03: migrations, 04: svc, 05: app wiring).

---

## Files

- Create: `internal/app/e2e_test.go`
- Create: `internal/app/testdb_helper_test.go`

---

### Task 1: Shared DB test helper

**Files:**
- Create: `internal/app/testdb_helper_test.go`

- [ ] **Step 1: Write the helper**

Create `internal/app/testdb_helper_test.go`:

```go
package app

import (
	"context"
	"testing"

	"github.com/4sp1/vipo-go/internal/port/state/sqlite"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	srv, err := NewServer(":memory:", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.DB.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { srv.Close() })
	return srv
}
```

This helper creates an in-memory SQLite DB, runs goose migrations, and returns a `*Server` with a real `http.Handler`. The `t.Cleanup` ensures the DB is closed after the test. [S18].

- [ ] **Step 2: Verify it compiles**

Run: `go test ./internal/app/ -run xxx -v`
Expected: PASS (no tests match `xxx`, but compilation succeeds)

- [ ] **Step 3: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
test(app): add shared in-memory DB test helper

setupTestServer creates in-memory SQLite, runs goose
migrations, returns *Server for e2e tests. [S18]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 2: E2E — Notes CRUD

**Files:**
- Create: `internal/app/e2e_test.go`

- [ ] **Step 1: Write the tests**

Create `internal/app/e2e_test.go`:

```go
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
)

// --- Helpers ---

func doRequest(t *testing.T, handler http.Handler, method, path string, body any) (*httptest.ResponseRecorder, []byte) {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w, w.Body.Bytes()
}

func decodeJSON(t *testing.T, b []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, b)
	}
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("status = %d, want %d; body: %s", w.Code, want, w.Body.Bytes())
	}
}

func assertCode(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var errResp oapi.Error
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("unmarshal error: %v\nbody: %s", err, body)
	}
	if errResp.Code == nil {
		t.Fatal("code is nil")
	}
	if *errResp.Code != wantCode {
		t.Errorf("code = %q, want %q", *errResp.Code, wantCode)
	}
}

// --- CreateNote ---

func TestE2ECreateNote(t *testing.T) {
	srv := setupTestServer(t)

	body := oapi.CreateNoteRequest{
		Note:          "test note",
		PomodoroState: oapi.Work,
	}
	w, respBody := doRequest(t, srv.Handler, http.MethodPost, "/notes", body)
	assertStatus(t, w, 201)

	var note oapi.Note
	decodeJSON(t, respBody, &note)
	if note.Note != "test note" {
		t.Errorf("Note = %q, want 'test note'", note.Note)
	}
	if note.PomodoroState != oapi.Work {
		t.Errorf("PomodoroState = %q, want %q", note.PomodoroState, oapi.Work)
	}
	if note.Id != 1 {
		t.Errorf("Id = %d, want 1", note.Id)
	}
	if note.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestE2ECreateNoteInvalidPomodoroState(t *testing.T) {
	srv := setupTestServer(t)

	body := oapi.CreateNoteRequest{
		Note:          "test",
		PomodoroState: "invalid",
	}
	w, respBody := doRequest(t, srv.Handler, http.MethodPost, "/notes", body)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_pomodoro_state")
}

func TestE2ECreateNoteMissingBody(t *testing.T) {
	srv := setupTestServer(t)

	w, _ := doRequest(t, srv.Handler, http.MethodPost, "/notes", nil)
	assertStatus(t, w, 400)
}

// --- GetNote ---

func TestE2EGetNote(t *testing.T) {
	srv := setupTestServer(t)

	// Create a note first
	createBody := oapi.CreateNoteRequest{Note: "fetch me", PomodoroState: oapi.ShortBreak}
	_, createResp := doRequest(t, srv.Handler, http.MethodPost, "/notes", createBody)
	var note oapi.Note
	decodeJSON(t, createResp, &note)

	// Get it
	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes/1", nil)
	assertStatus(t, w, 200)

	var got oapi.Note
	decodeJSON(t, respBody, &got)
	if got.Note != "fetch me" {
		t.Errorf("Note = %q, want 'fetch me'", got.Note)
	}
	if got.PomodoroState != oapi.ShortBreak {
		t.Errorf("PomodoroState = %q, want %q", got.PomodoroState, oapi.ShortBreak)
	}
}

func TestE2EGetNoteNotFound(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes/999", nil)
	assertStatus(t, w, 404)
	assertCode(t, respBody, "note_not_found")
}

// --- DeleteNote ---

func TestE2EDeleteNote(t *testing.T) {
	srv := setupTestServer(t)

	// Create
	createBody := oapi.CreateNoteRequest{Note: "delete me", PomodoroState: oapi.Work}
	doRequest(t, srv.Handler, http.MethodPost, "/notes", createBody)

	// Delete
	w, _ := doRequest(t, srv.Handler, http.MethodDelete, "/notes/1", nil)
	assertStatus(t, w, 204)

	// Verify gone
	w, _ = doRequest(t, srv.Handler, http.MethodGet, "/notes/1", nil)
	assertStatus(t, w, 404)
}

func TestE2EDeleteNoteNotFound(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodDelete, "/notes/999", nil)
	assertStatus(t, w, 404)
	assertCode(t, respBody, "note_not_found")
}

// --- ListNotes ---

func TestE2EListNotes(t *testing.T) {
	srv := setupTestServer(t)

	for i := 0; i < 7; i++ {
		createBody := oapi.CreateNoteRequest{Note: "note", PomodoroState: oapi.Work}
		doRequest(t, srv.Handler, http.MethodPost, "/notes", createBody)
	}

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes?limit=3&offset=0", nil)
	assertStatus(t, w, 200)

	var list oapi.NoteList
	decodeJSON(t, respBody, &list)
	if list.Notes == nil {
		t.Fatal("Notes is nil")
	}
	if len(*list.Notes) != 3 {
		t.Errorf("len(Notes) = %d, want 3", len(*list.Notes))
	}
	if list.Total == nil {
		t.Fatal("Total is nil")
	}
	if *list.Total != 7 {
		t.Errorf("Total = %d, want 7", *list.Total)
	}
}

func TestE2EListNotesDefaultLimit(t *testing.T) {
	srv := setupTestServer(t)

	for i := 0; i < 5; i++ {
		createBody := oapi.CreateNoteRequest{Note: "note", PomodoroState: oapi.Work}
		doRequest(t, srv.Handler, http.MethodPost, "/notes", createBody)
	}

	// No limit/offset → defaults: limit=50, offset=0
	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes", nil)
	assertStatus(t, w, 200)

	var list oapi.NoteList
	decodeJSON(t, respBody, &list)
	if len(*list.Notes) != 5 {
		t.Errorf("len(Notes) = %d, want 5", len(*list.Notes))
	}
	if *list.Total != 5 {
		t.Errorf("Total = %d, want 5", *list.Total)
	}
}

func TestE2EListNotesByPomodoroState(t *testing.T) {
	srv := setupTestServer(t)

	doRequest(t, srv.Handler, http.MethodPost, "/notes", oapi.CreateNoteRequest{Note: "work", PomodoroState: oapi.Work})
	doRequest(t, srv.Handler, http.MethodPost, "/notes", oapi.CreateNoteRequest{Note: "break", PomodoroState: oapi.ShortBreak})
	doRequest(t, srv.Handler, http.MethodPost, "/notes", oapi.CreateNoteRequest{Note: "work2", PomodoroState: oapi.Work})

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes?pomodoro_state=work", nil)
	assertStatus(t, w, 200)

	var list oapi.NoteList
	decodeJSON(t, respBody, &list)
	if len(*list.Notes) != 2 {
		t.Errorf("len(Notes) = %d, want 2", len(*list.Notes))
	}
	if *list.Total != 2 {
		t.Errorf("Total = %d, want 2", *list.Total)
	}
}

func TestE2EListNotesInvalidLimit(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes?limit=0", nil)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_limit")
}

func TestE2EListNotesLimitTooLarge(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes?limit=101", nil)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_limit")
}

func TestE2EListNotesInvalidOffset(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes?offset=-1", nil)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_offset")
}

func TestE2EListNotesEmpty(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/notes", nil)
	assertStatus(t, w, 200)

	var list oapi.NoteList
	decodeJSON(t, respBody, &list)
	if list.Notes == nil {
		t.Error("Notes is nil, want empty array")
	}
	if len(*list.Notes) != 0 {
		t.Errorf("len(Notes) = %d, want 0", len(*list.Notes))
	}
	if list.Total == nil || *list.Total != 0 {
		t.Errorf("Total = %v, want 0", list.Total)
	}
}

// suppress unused import
var _ = context.Background
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/app/ -run TestE2E -v`
Expected: PASS

- [ ] **Step 3: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
test(app): e2e notes CRUD

CreateNote, GetNote, DeleteNote, ListNotes with
pagination, filters, defaults, validation (400/404),
and error code assertions. [S18]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 3: E2E — Logs CRUD

**Files:**
- Modify: `internal/app/e2e_test.go`

- [ ] **Step 1: Write the tests**

Append to `internal/app/e2e_test.go`:

```go
// --- CreateLogEntry ---

func TestE2ECreateLogEntry(t *testing.T) {
	srv := setupTestServer(t)

	payload := map[string]any{"key": "value"}
	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: oapi.Work,
		Payload: &payload,
	}
	w, respBody := doRequest(t, srv.Handler, http.MethodPost, "/logs", body)
	assertStatus(t, w, 201)

	var entry oapi.LogEntry
	decodeJSON(t, respBody, &entry)
	if entry.Id != 1 {
		t.Errorf("Id = %d, want 1", entry.Id)
	}
	if entry.Action != oapi.Start {
		t.Errorf("Action = %q, want %q", entry.Action, oapi.Start)
	}
	if entry.Session != oapi.Work {
		t.Errorf("Session = %q, want %q", entry.Session, oapi.Work)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
	if entry.Payload == nil {
		t.Error("Payload is nil")
	}
}

func TestE2ECreateLogEntryNilPayload(t *testing.T) {
	srv := setupTestServer(t)

	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: oapi.Work,
	}
	w, respBody := doRequest(t, srv.Handler, http.MethodPost, "/logs", body)
	assertStatus(t, w, 201)

	var entry oapi.LogEntry
	decodeJSON(t, respBody, &entry)
	if entry.Payload != nil {
		t.Errorf("Payload = %v, want nil", entry.Payload)
	}
}

func TestE2ECreateLogEntryInvalidAction(t *testing.T) {
	srv := setupTestServer(t)

	body := oapi.CreateLogEntryRequest{
		Action:  "invalid",
		Session: oapi.Work,
	}
	w, respBody := doRequest(t, srv.Handler, http.MethodPost, "/logs", body)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_log_action")
}

func TestE2ECreateLogEntryInvalidSession(t *testing.T) {
	srv := setupTestServer(t)

	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: "invalid",
	}
	w, respBody := doRequest(t, srv.Handler, http.MethodPost, "/logs", body)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_pomodoro_state")
}

// --- ListLogEntries ---

func TestE2EListLogEntries(t *testing.T) {
	srv := setupTestServer(t)

	for i := 0; i < 5; i++ {
		body := oapi.CreateLogEntryRequest{
			Action:  oapi.Start,
			Session: oapi.Work,
		}
		doRequest(t, srv.Handler, http.MethodPost, "/logs", body)
	}

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs?limit=3&offset=0", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, respBody, &list)
	if list.Entries == nil {
		t.Fatal("Entries is nil")
	}
	if len(*list.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(*list.Entries))
	}
	if list.Total == nil || *list.Total != 5 {
		t.Errorf("Total = %v, want 5", list.Total)
	}
}

func TestE2EListLogEntriesDefaultLimit(t *testing.T) {
	srv := setupTestServer(t)

	for i := 0; i < 3; i++ {
		body := oapi.CreateLogEntryRequest{
			Action:  oapi.Start,
			Session: oapi.Work,
		}
		doRequest(t, srv.Handler, http.MethodPost, "/logs", body)
	}

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, respBody, &list)
	if len(*list.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(*list.Entries))
	}
	if *list.Total != 3 {
		t.Errorf("Total = %d, want 3", *list.Total)
	}
}

func TestE2EListLogEntriesFilterByAction(t *testing.T) {
	srv := setupTestServer(t)

	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.Work})
	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Pause, Session: oapi.Work})

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs?action=start", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, respBody, &list)
	if len(*list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*list.Entries))
	}
	if *list.Total != 1 {
		t.Errorf("Total = %d, want 1", *list.Total)
	}
}

func TestE2EListLogEntriesFilterBySession(t *testing.T) {
	srv := setupTestServer(t)

	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.Work})
	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.ShortBreak})

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs?session=work", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, respBody, &list)
	if len(*list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*list.Entries))
	}
}

func TestE2EListLogEntriesFilterByTimeRange(t *testing.T) {
	srv := setupTestServer(t)

	// Create entries — timestamps are set by svc to time.Now().UTC()
	// We can't control exact timestamps, so we use a wide range
	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.Work})
	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.Work})

	// Use from=2000 and to=2100 to catch everything
	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs?from=2000-01-01T00:00:00Z&to=2100-01-01T00:00:00Z", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, respBody, &list)
	if len(*list.Entries) != 2 {
		t.Errorf("len(Entries) = %d, want 2", len(*list.Entries))
	}
}

func TestE2EListLogEntriesInvalidLimit(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs?limit=0", nil)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_limit")
}

func TestE2EListLogEntriesInvalidOffset(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs?offset=-1", nil)
	assertStatus(t, w, 400)
	assertCode(t, respBody, "invalid_offset")
}

func TestE2EListLogEntriesEmpty(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, respBody, &list)
	if list.Entries == nil {
		t.Error("Entries is nil, want empty array")
	}
	if len(*list.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(*list.Entries))
	}
	if list.Total == nil || *list.Total != 0 {
		t.Errorf("Total = %v, want 0", list.Total)
	}
}

func TestE2ELogEntryPayloadRoundTrip(t *testing.T) {
	srv := setupTestServer(t)

	original := map[string]any{
		"string":  "hello",
		"number":  float64(42),
		"nested":  map[string]any{"inner": "value"},
		"array":   []any{float64(1), float64(2), float64(3)},
		"unicode": "héllo wörld",
	}
	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: oapi.Work,
		Payload: &original,
	}
	_, createResp := doRequest(t, srv.Handler, http.MethodPost, "/logs", body)
	var entry oapi.LogEntry
	decodeJSON(t, createResp, &entry)

	// List and verify payload round-trips
	w, listResp := doRequest(t, srv.Handler, http.MethodGet, "/logs", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, listResp, &list)
	if list.Entries == nil || len(*list.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %v", list.Entries)
	}

	got := (*list.Entries)[0]
	if got.Payload == nil {
		t.Fatal("Payload is nil")
	}

	origJSON, _ := json.Marshal(original)
	gotJSON, _ := json.Marshal(*got.Payload)
	if string(origJSON) != string(gotJSON) {
		t.Errorf("payload mismatch:\n  orig: %s\n  got:  %s", origJSON, gotJSON)
	}
}

func TestE2EListLogEntriesCombinedFilters(t *testing.T) {
	srv := setupTestServer(t)

	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Pause, Session: oapi.Work})
	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.Work})
	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Pause, Session: oapi.ShortBreak})

	w, respBody := doRequest(t, srv.Handler, http.MethodGet, "/logs?action=pause&session=work", nil)
	assertStatus(t, w, 200)

	var list oapi.LogEntryList
	decodeJSON(t, respBody, &list)
	if len(*list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*list.Entries))
	}
	if *list.Total != 1 {
		t.Errorf("Total = %d, want 1", *list.Total)
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/app/ -run TestE2E -v`
Expected: PASS

- [ ] **Step 3: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
test(app): e2e logs CRUD + payload round-trip

CreateLogEntry, ListLogEntries with action/session/
time filters, pagination, validation, payload JSON
round-trip assertions. [S6, S7, S18]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 4: Svc-level unit tests for validation edge cases

**Files:**
- Modify: `internal/svc/service_test.go`

These tests cover edge cases not easily tested via e2e: exact error types, boundary conditions, and the `from`/`to` truncation policy. [S5].

- [ ] **Step 1: Write the tests**

Append to `internal/svc/service_test.go`:

```go
func TestListLogEntriesFromTruncatesToSecond(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	// Add an entry at a specific second
	logs.AddEntry(context.Background(), domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC),
	})

	// from with sub-second → should floor to the second
	from := time.Date(2026, 1, 1, 0, 0, 30, 500000000, time.UTC)
	to := time.Date(2026, 1, 1, 23, 59, 59, 0, time.UTC)

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{From: &from, To: &to},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1 (from truncated to second)", len(*listResp.Entries))
	}
}

func TestListLogEntriesToCeilsToSecond(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	// Entry at exactly 00:00:31
	logs.AddEntry(context.Background(), domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 31, 0, time.UTC),
	})

	// to with sub-second at 00:00:30.5 → should ceil to 00:00:31
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 30, 500000000, time.UTC)

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{From: &from, To: &to},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1 (to ceiled to second)", len(*listResp.Entries))
	}
}

func TestListLogEntriesBoundaryExactSecond(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	// Entry at exactly 00:00:30
	logs.AddEntry(context.Background(), domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC),
	})

	// from=00:00:30, to=00:00:30 → inclusive, should find the entry
	from := time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC)

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{From: &from, To: &to},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1 (inclusive boundary)", len(*listResp.Entries))
	}
}

func TestListNotesLimitBoundary(t *testing.T) {
	s := New(newRecordingNoteStore(), &stubEventLog{})

	// limit=1 (minimum)
	limit := 1
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok := resp.(oapi.ListNotes200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}

	// limit=100 (maximum)
	limit = 100
	resp, err = s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok = resp.(oapi.ListNotes200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
}

func TestListLogEntriesLimitBoundary(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog{})

	limit := 1
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}

	limit = 100
	resp, err = s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok = resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
}
```

Note: These tests assume the svc returns response objects (sub-plan 04 approach). If the svc was updated to return errors (sub-plan 05, Task 3), adjust assertions to check `err != nil` for validation failures and `resp` type for success.

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/svc/ -run "TestListLog|TestListNotes" -v`
Expected: PASS

- [ ] **Step 3: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
test(svc): validation edge cases for from/to and
limits

from truncates to second, to ceils, inclusive boundary.
Limit boundaries 1 and 100. [S5, S15]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 5: Full test suite run

**Files:**
- None (verification only)

- [ ] **Step 1: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Run vet**

Run: `go vet ./...`
Expected: PASS

- [ ] **Step 3: Run build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
test: verify full test suite passes

All packages: domain, adapter, port, svc, app, cmd.

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 6: File issue 26 for payload future work

**Files:**
- Create: `docs/issues/26.md`

- [ ] **Step 1: Write the issue**

Create `docs/issues/26.md`:

```markdown
# Investigate payload semantics via e2e + update OAPI spec

The OAPI spec's `payload` field (on `CreateLogEntryRequest`
and `LogEntry`) is a free-form object with no schema
constraints beyond "Optional action-specific data." The
current implementation uses `map[string]any` with
`json.Marshal`/`json.Unmarshal` round-trip (semantically
lossless, not byte-faithful — see Gap 2 in
`docs/drafts/20261012_openapi_gaps.md`).

After the e2e test suite from the wire-http-server plan is
exercised against the Flutter client, revisit:

1. What does the Flutter client actually send in `payload`?
2. Does the spec need per-action payload schemas?
3. Should the domain layer decouple from `map[string]any` to
   a typed payload (revisiting the sealed `Message` interface
   approach)?

References: [S11] in
`docs/drafts/20261021_wire_http_server_decisions.md`,
Gap 2 in `docs/drafts/20261012_openapi_gaps.md`.
```

- [ ] **Step 2: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
docs(issues): file #26 for payload semantics

Deferred from wire-http-server plan [S11]. Track
payload spec investigation post-e2e. [S22]

Refs #22, #26
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`
