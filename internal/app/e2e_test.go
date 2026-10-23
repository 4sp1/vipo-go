package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
)

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

func TestE2EGetNote(t *testing.T) {
	srv := setupTestServer(t)

	createBody := oapi.CreateNoteRequest{Note: "fetch me", PomodoroState: oapi.ShortBreak}
	_, createResp := doRequest(t, srv.Handler, http.MethodPost, "/notes", createBody)
	var note oapi.Note
	decodeJSON(t, createResp, &note)

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

func TestE2EDeleteNote(t *testing.T) {
	srv := setupTestServer(t)

	createBody := oapi.CreateNoteRequest{Note: "delete me", PomodoroState: oapi.Work}
	doRequest(t, srv.Handler, http.MethodPost, "/notes", createBody)

	w, _ := doRequest(t, srv.Handler, http.MethodDelete, "/notes/1", nil)
	assertStatus(t, w, 204)

	w, _ = doRequest(t, srv.Handler, http.MethodGet, "/notes/1", nil)
	assertStatus(t, w, 404)
}

func TestE2EDeleteNoteNotFound(t *testing.T) {
	srv := setupTestServer(t)

	w, respBody := doRequest(t, srv.Handler, http.MethodDelete, "/notes/999", nil)
	assertStatus(t, w, 404)
	assertCode(t, respBody, "note_not_found")
}

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

	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.Work})
	doRequest(t, srv.Handler, http.MethodPost, "/logs", oapi.CreateLogEntryRequest{Action: oapi.Start, Session: oapi.Work})

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
