# 04 — Service Layer

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the service layer that satisfies `oapi.StrictServerInterface` — six handlers covering notes CRUD and log CRUD, with translation between OAPI string enums and domain int enums, limit/offset defaulting and validation, and error mapping to typed HTTP error codes.

**Architecture:** `internal/svc/` package. A `Service` struct holds `note.Store` and `log.EventLog` dependencies. It implements all six `StrictServerInterface` methods. The svc is the time authority (`CreatedAt = time.Now().UTC()`) and the port is the id authority. The svc does NOT know about HTTP error shaping — it returns adapter errors (`note.ErrNotFound`, `log.ErrNotFound`) and validation errors; the app layer (sub-plan 05) maps them to `apperr.Code` → `Error{code, message}`. [S8], [S16], [S12], [S15/S24].

**Tech Stack:** Go stdlib (`context`, `errors`, `fmt`, `time`), `oapi-codegen` generated types + strict server interface, domain packages, adapter packages, `port/oapi/http` translation layer.

**Depends on:** Sub-plan 01 (adapter interfaces, updated `log.Entry`), sub-plan 02 (SQLite ports implementing those interfaces).

---

## Files

- Create: `internal/svc/service.go`
- Create: `internal/svc/service_test.go`

---

### Task 1: Service struct and constructor

**Files:**
- Create: `internal/svc/service.go`
- Test: `internal/svc/service_test.go`

- [ ] **Step 1: Write the failing test**

```go
package svc

import (
	"testing"

	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
)

func TestNewService(t *testing.T) {
	s := New(&stubNoteStore{}, &stubEventLog{})
	if s == nil {
		t.Fatal("New returned nil")
	}
}

type stubNoteStore struct {
	adaptNote.Store
}

type stubEventLog struct {
	adaptLog.EventLog
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/svc/ -run TestNewService -v`
Expected: FAIL — `undefined: New`, `undefined: Service`

- [ ] **Step 3: Write minimal implementation**

Create `internal/svc/service.go`:

```go
package svc

import (
	"context"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
)

type Service struct {
	notes adaptNote.Store
	logs  adaptLog.EventLog
}

func New(notes adaptNote.Store, logs adaptLog.EventLog) *Service {
	return &Service{notes: notes, logs: logs}
}

// Compile-time check: Service satisfies StrictServerInterface.
var _ oapi.StrictServerInterface = (*Service)(nil)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/svc/ -run TestNewService -v`
Expected: PASS (the compile-time check in service.go will fail if methods are missing — this is expected; we add them in subsequent tasks)

Note: The `var _ oapi.StrictServerInterface = (*Service)(nil)` line will cause a compile error until all six methods are implemented. Temporarily comment it out and re-enable after Task 6. Or, implement all six method stubs now and fill them in tasks 2-6. For the TDD flow, comment it out for now:

Replace the line:
```go
var _ oapi.StrictServerInterface = (*Service)(nil)
```
with:
```go
// var _ oapi.StrictServerInterface = (*Service)(nil) // enabled after all methods implemented
```

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(svc): add Service struct with note/log dependencies

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 2: CreateNote handler

**Files:**
- Modify: `internal/svc/service.go`
- Test: `internal/svc/service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/svc/service_test.go`:

```go
import (
	"context"
	"errors"
	"testing"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
)

func TestCreateNote(t *testing.T) {
	tests := []struct {
		name    string
		body    oapi.CreateNoteRequest
		wantErr bool
		wantNote string
	}{
		{
			name:     "valid work note",
			body:     oapi.CreateNoteRequest{Note: "hello", PomodoroState: oapi.Work},
			wantNote: "hello",
		},
		{
			name:     "valid short_break note",
			body:     oapi.CreateNoteRequest{Note: "break time", PomodoroState: oapi.ShortBreak},
			wantNote: "break time",
		},
		{
			name:    "invalid pomodoro state",
			body:    oapi.CreateNoteRequest{Note: "test", PomodoroState: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &recordingNoteStore{}
			s := New(store, &stubEventLog{})

			resp, err := s.CreateNote(context.Background(), oapi.CreateNoteRequestObject{
				Body: &tt.body,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateNote: %v", err)
			}

			createResp, ok := resp.(oapi.CreateNote201JSONResponse)
			if !ok {
				t.Fatalf("unexpected response type: %T", resp)
			}
			if createResp.Note != tt.wantNote {
				t.Errorf("Note = %q, want %q", createResp.Note, tt.wantNote)
			}
			if createResp.PomodoroState != tt.body.PomodoroState {
				t.Errorf("PomodoroState = %q, want %q", createResp.PomodoroState, tt.body.PomodoroState)
			}
			if createResp.Id != 1 {
				t.Errorf("Id = %d, want 1", createResp.Id)
			}
		})
	}
}

type recordingNoteStore struct {
	notes map[domnote.ID]domnote.State
	nextID domnote.ID
}

func newRecordingNoteStore() *recordingNoteStore {
	return &recordingNoteStore{notes: make(map[domnote.ID]domnote.State), nextID: 1}
}

func (s *recordingNoteStore) AddNote(_ context.Context, n domnote.State) (domnote.ID, error) {
	id := s.nextID
	s.nextID++
	n.ID = id
	s.notes[id] = n
	return id, nil
}

func (s *recordingNoteStore) ListNotes(_ context.Context, _ adaptNote.ListParams) (adaptNote.List, error) {
	notes := make([]domnote.State, 0, len(s.notes))
	for _, n := range s.notes {
		notes = append(notes, n)
	}
	return adaptNote.List{Notes: notes, Total: len(notes)}, nil
}

func (s *recordingNoteStore) GetNote(_ context.Context, id domnote.ID) (domnote.State, error) {
	n, ok := s.notes[id]
	if !ok {
		return domnote.State{}, adaptNote.ErrNotFound
	}
	return n, nil
}

func (s *recordingNoteStore) DeleteNote(_ context.Context, id domnote.ID) error {
	if _, ok := s.notes[id]; !ok {
		return adaptNote.ErrNotFound
	}
	delete(s.notes, id)
	return nil
}
```

Also update the stub from Task 1 — replace `stubNoteStore` with the real `recordingNoteStore`:

```go
// Remove the old stubNoteStore embedding — recordingNoteStore replaces it.
// stubEventLog stays for now.
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/svc/ -run TestCreateNote -v`
Expected: FAIL — `undefined: Service.CreateNote`

- [ ] **Step 3: Write minimal implementation**

Add to `internal/svc/service.go`:

```go
import (
	"context"
	"fmt"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
	"github.com/4sp1/vipo-go/internal/port/oapi/http"
)

func (s *Service) CreateNote(ctx context.Context, req oapi.CreateNoteRequestObject) (oapi.CreateNoteResponseObject, error) {
	if req.Body == nil {
		return oapi.CreateNote400JSONResponse{Message: "missing request body"}, nil
	}

	ps, err := http.PomodoroStateToDomain(req.Body.PomodoroState)
	if err != nil {
		return oapi.CreateNote400JSONResponse{
			Message: fmt.Sprintf("invalid pomodoro_state: %s", req.Body.PomodoroState),
		}, nil
	}

	n := domnote.State{
		Note:      req.Body.Note,
		Pomodoro:  ps,
		CreatedAt: time.Now().UTC(),
	}

	id, err := s.notes.AddNote(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("add note: %w", err)
	}

	respPS, err := http.PomodoroStateFromDomain(ps)
	if err != nil {
		return nil, fmt.Errorf("convert pomodoro state: %w", err)
	}

	return oapi.CreateNote201JSONResponse{
		Id:            int64(id),
		Note:          n.Note,
		PomodoroState: respPS,
		CreatedAt:     n.CreatedAt,
	}, nil
}
```

Key decisions: svc sets `CreatedAt = time.Now().UTC()` — svc is time-authority. [S12]. Port returns the autoincremented id — port is id-authority. Validation errors return `CreateNote400JSONResponse` (HTTP 400), not Go errors — the svc decides 400 vs 500. [S8]. The `code` field on the 400 response is NOT set here — that is the app layer's job (sub-plan 05). [S16].

Note: the 400 response currently omits the `Code` field. Sub-plan 05 wraps the svc and injects `Code` values. If the app layer approach instead post-processes the response, the svc returns `Error{Message: ...}` and the app layer enriches. For now, the svc returns the response without `Code`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/svc/ -run TestCreateNote -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(svc): implement CreateNote handler

Validates pomodoro_state, sets CreatedAt to UTC now,
calls note.Store.AddNote, returns 201 with Note. [S8,
S12]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 3: GetNote and DeleteNote handlers

**Files:**
- Modify: `internal/svc/service.go`
- Test: `internal/svc/service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/svc/service_test.go`:

```go
func TestGetNote(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	n := domnote.State{
		Note:      "fetch me",
		Pomodoro:  pomodoro.LongBreak,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, _ := store.AddNote(context.Background(), n)

	resp, err := s.GetNote(context.Background(), oapi.GetNoteRequestObject{Id: int64(id)})
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}

	getResp, ok := resp.(oapi.GetNote200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if getResp.Note != "fetch me" {
		t.Errorf("Note = %q, want 'fetch me'", getResp.Note)
	}
	if getResp.PomodoroState != oapi.LongBreak {
		t.Errorf("PomodoroState = %q, want %q", getResp.PomodoroState, oapi.LongBreak)
	}
}

func TestGetNoteNotFound(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	resp, err := s.GetNote(context.Background(), oapi.GetNoteRequestObject{Id: 999})
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}

	_, ok := resp.(oapi.GetNote404JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T, want GetNote404JSONResponse", resp)
	}
}

func TestDeleteNote(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	n := domnote.State{
		Note:     "delete me",
		Pomodoro: pomodoro.Work,
	}
	id, _ := store.AddNote(context.Background(), n)

	resp, err := s.DeleteNote(context.Background(), oapi.DeleteNoteRequestObject{Id: int64(id)})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, ok := resp.(oapi.DeleteNote204Response)
	if !ok {
		t.Fatalf("unexpected response type: %T, want DeleteNote204Response", resp)
	}

	// Verify it's gone
	_, err = store.GetNote(context.Background(), id)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("GetNote after delete: %v, want ErrNotFound", err)
	}
}

func TestDeleteNoteNotFound(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	resp, err := s.DeleteNote(context.Background(), oapi.DeleteNoteRequestObject{Id: 999})
	if err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, ok := resp.(oapi.DeleteNote404JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T, want DeleteNote404JSONResponse", resp)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/svc/ -run "TestGetNote|TestDeleteNote" -v`
Expected: FAIL — `undefined: Service.GetNote`, `undefined: Service.DeleteNote`

- [ ] **Step 3: Write minimal implementation**

Add to `internal/svc/service.go`:

```go
func (s *Service) GetNote(ctx context.Context, req oapi.GetNoteRequestObject) (oapi.GetNoteResponseObject, error) {
	n, err := s.notes.GetNote(ctx, domnote.ID(req.Id))
	if err != nil {
		if errors.Is(err, adaptNote.ErrNotFound) {
			return oapi.GetNote404JSONResponse{Message: "note not found"}, nil
		}
		return nil, fmt.Errorf("get note: %w", err)
	}

	respPS, err := http.PomodoroStateFromDomain(n.Pomodoro)
	if err != nil {
		return nil, fmt.Errorf("convert pomodoro state: %w", err)
	}

	return oapi.GetNote200JSONResponse{
		Id:            int64(n.ID),
		Note:          n.Note,
		PomodoroState: respPS,
		CreatedAt:     n.CreatedAt,
	}, nil
}

func (s *Service) DeleteNote(ctx context.Context, req oapi.DeleteNoteRequestObject) (oapi.DeleteNoteResponseObject, error) {
	err := s.notes.DeleteNote(ctx, domnote.ID(req.Id))
	if err != nil {
		if errors.Is(err, adaptNote.ErrNotFound) {
			return oapi.DeleteNote404JSONResponse{Message: "note not found"}, nil
		}
		return nil, fmt.Errorf("delete note: %w", err)
	}

	return oapi.DeleteNote204Response{}, nil
}
```

The svc maps `adaptNote.ErrNotFound` to 404 JSON responses — no Go error propagated to the app layer for not-found. Other errors (DB failures) propagate as Go errors to be handled by the app layer's `ResponseErrorHandlerFunc`. [S3], [S8].

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/svc/ -run "TestGetNote|TestDeleteNote" -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(svc): implement GetNote and DeleteNote handlers

GetNote returns 404 on ErrNotFound, 200 on success.
DeleteNote returns 404 on ErrNotFound, 204 on success.
[S3, S8]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 4: ListNotes handler with limit/offset defaults and validation

**Files:**
- Modify: `internal/svc/service.go`
- Test: `internal/svc/service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/svc/service_test.go`:

```go
func TestListNotesDefaultLimitOffset(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	for i := 0; i < 60; i++ {
		store.AddNote(context.Background(), domnote.State{
			Note:     "note",
			Pomodoro: pomodoro.Work,
		})
	}

	// nil limit and offset → defaults: limit=50, offset=0
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	listResp, ok := resp.(oapi.ListNotes200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if listResp.Notes == nil {
		t.Fatal("Notes is nil")
	}
	if len(*listResp.Notes) != 50 {
		t.Errorf("len(Notes) = %d, want 50", len(*listResp.Notes))
	}
	if listResp.Total == nil {
		t.Fatal("Total is nil")
	}
	if *listResp.Total != 60 {
		t.Errorf("Total = %d, want 60", *listResp.Total)
	}
}

func TestListNotesInvalidLimit(t *testing.T) {
	s := New(newRecordingNoteStore(), &stubEventLog{})

	// limit < 1
	badLimit := 0
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Limit: &badLimit},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok := resp.(oapi.ListNotes400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for limit=0, got %T", resp)
	}

	// limit > 100
	bigLimit := 101
	resp, err = s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Limit: &bigLimit},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok = resp.(oapi.ListNotes400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for limit=101, got %T", resp)
	}
}

func TestListNotesInvalidOffset(t *testing.T) {
	s := New(newRecordingNoteStore(), &stubEventLog{})

	badOffset := -1
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{Offset: &badOffset},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok := resp.(oapi.ListNotes400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for offset=-1, got %T", resp)
	}
}

func TestListNotesByPomodoroState(t *testing.T) {
	store := newRecordingNoteStore()
	s := New(store, &stubEventLog{})

	store.AddNote(context.Background(), domnote.State{Note: "work", Pomodoro: pomodoro.Work})
	store.AddNote(context.Background(), domnote.State{Note: "break", Pomodoro: pomodoro.ShortBreak})
	store.AddNote(context.Background(), domnote.State{Note: "work2", Pomodoro: pomodoro.Work})

	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{PomodoroState: &[]oapi.PomodoroState{oapi.Work}[0]},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	listResp, ok := resp.(oapi.ListNotes200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Notes) != 2 {
		t.Errorf("len(Notes) = %d, want 2", len(*listResp.Notes))
	}
	if *listResp.Total != 2 {
		t.Errorf("Total = %d, want 2", *listResp.Total)
	}
}

func TestListNotesInvalidPomodoroState(t *testing.T) {
	s := New(newRecordingNoteStore(), &stubEventLog{})

	badState := oapi.PomodoroState("invalid")
	resp, err := s.ListNotes(context.Background(), oapi.ListNotesRequestObject{
		Params: oapi.ListNotesParams{PomodoroState: &badState},
	})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	_, ok := resp.(oapi.ListNotes400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for invalid pomodoro_state, got %T", resp)
	}
}
```

Note: `ListNotes` in the OAPI spec has only a 200 response defined, no 400. The generated code has no `ListNotes400JSONResponse` type. Since the spec defines no 400 for `listNotes`, validation failures (invalid limit, offset, pomodoro_state) must still return a 400 — we need to construct the response manually. Let's check: the OAPI spec for `GET /notes` only defines a 200 response. The generated code only has `ListNotes200JSONResponse`. So we need to write a raw response writer for 400s.

Add a helper type in `service.go`:

```go
type listNotes400Response struct {
	message string
}

func (r listNotes400Response) VisitListNotesResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	return json.NewEncoder(w).Encode(oapi.Error{Message: r.message})
}
```

This implements `ListNotesResponseObject` by writing a 400 status with an `Error` body, matching the spec's `Error` schema even though the spec doesn't explicitly define a 400 for this endpoint.

Wait — there's a conflict: `service.go` already imports `http` as the translation package (`port/oapi/http`). The `net/http` import would conflict. The test imports use `oapi` for the generated types. We need to alias one of them.

Resolution: In `service.go`, alias the translation package:

```go
import (
	nethttp "net/http"
	"encoding/json"
	// ...
	xlate "github.com/4sp1/vipo-go/internal/port/oapi/http"
)
```

And use `xlate.PomodoroStateToDomain` etc. throughout. The 400 response uses `nethttp.ResponseWriter`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/svc/ -run TestListNotes -v`
Expected: FAIL — `undefined: Service.ListNotes`

- [ ] **Step 3: Write minimal implementation**

First, update the imports in `internal/svc/service.go`. Replace the import block with:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	nethttp "net/http"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
	xlate "github.com/4sp1/vipo-go/internal/port/oapi/http"
)
```

Note: update all existing `http.PomodoroStateToDomain` references to `xlate.PomodoroStateToDomain`, etc.

Add a 400 response helper and the `ListNotes` method:

```go
type listNotes400Response struct {
	message string
}

func (r listNotes400Response) VisitListNotesResponse(w nethttp.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	return json.NewEncoder(w).Encode(oapi.Error{Message: r.message})
}

func (s *Service) ListNotes(ctx context.Context, req oapi.ListNotesRequestObject) (oapi.ListNotesResponseObject, error) {
	// Default limit=50, offset=0 [S15/S24]
	limit := 50
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	offset := 0
	if req.Params.Offset != nil {
		offset = *req.Params.Offset
	}

	// Validate [S15/S24]
	if limit < 1 || limit > 100 {
		return listNotes400Response{message: "limit must be between 1 and 100"}, nil
	}
	if offset < 0 {
		return listNotes400Response{message: "offset must be >= 0"}, nil
	}

	// Translate pomodoro_state filter
	var psFilter *pomodoro.State
	if req.Params.PomodoroState != nil {
		ps, err := xlate.PomodoroStateToDomain(*req.Params.PomodoroState)
		if err != nil {
			return listNotes400Response{message: fmt.Sprintf("invalid pomodoro_state: %s", *req.Params.PomodoroState)}, nil
		}
		psFilter = &ps
	}

	list, err := s.notes.ListNotes(ctx, adaptNote.ListParams{
		PomodoroState: psFilter,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	notes := make([]oapi.Note, 0, len(list.Notes))
	for _, n := range list.Notes {
		respPS, err := xlate.PomodoroStateFromDomain(n.Pomodoro)
		if err != nil {
			return nil, fmt.Errorf("convert pomodoro state: %w", err)
		}
		notes = append(notes, oapi.Note{
			Id:            int64(n.ID),
			Note:          n.Note,
			PomodoroState: respPS,
			CreatedAt:     n.CreatedAt,
		})
	}

	total := list.Total
	return oapi.ListNotes200JSONResponse{
		Notes: &notes,
		Total: &total,
	}, nil
}
```

Key decisions: nil `limit` defaults to 50, nil `offset` defaults to 0. [S15/S24]. Out-of-range `limit` (<1 or >100) and negative `offset` return 400. [S15/S24]. Defaults are applied in the svc; the port receives validated `int`. The 400 response uses a custom `listNotes400Response` because the OAPI spec defines no 400 for `listNotes` — the generated code has no `ListNotes400JSONResponse` type. The response still returns the `Error` schema (`{message}`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/svc/ -run TestListNotes -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(svc): implement ListNotes with validation

Default limit=50 offset=0. Reject limit<1 or >100,
offset<0 as 400. Translate pomodoro_state filter.
[S15]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 5: CreateLogEntry handler

**Files:**
- Modify: `internal/svc/service.go`
- Test: `internal/svc/service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/svc/service_test.go`:

```go
type recordingEventLog struct {
	entries map[domlog.ID]domlog.Entry
	nextID  domlog.ID
}

func newRecordingEventLog() *recordingEventLog {
	return &recordingEventLog{entries: make(map[domlog.ID]domlog.Entry), nextID: 1}
}

func (e *recordingEventLog) AddEntry(_ context.Context, entry domlog.Entry) (domlog.ID, error) {
	id := e.nextID
	e.nextID++
	entry.ID = id
	e.entries[id] = entry
	return id, nil
}

func (e *recordingEventLog) ListEntries(_ context.Context, _ adaptLog.ListParams) (adaptLog.EntryList, error) {
	entries := make([]domlog.Entry, 0, len(e.entries))
	for _, e := range e.entries {
		entries = append(entries, e)
	}
	return adaptLog.EntryList{Entries: entries, Total: len(entries)}, nil
}

func TestCreateLogEntry(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	payload := map[string]any{"key": "value"}
	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: oapi.Work,
		Payload: &payload,
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}

	createResp, ok := resp.(oapi.CreateLogEntry201JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if createResp.Id != 1 {
		t.Errorf("Id = %d, want 1", createResp.Id)
	}
	if createResp.Action != oapi.Start {
		t.Errorf("Action = %q, want %q", createResp.Action, oapi.Start)
	}
	if createResp.Session != oapi.Work {
		t.Errorf("Session = %q, want %q", createResp.Session, oapi.Work)
	}
}

func TestCreateLogEntryInvalidAction(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	body := oapi.CreateLogEntryRequest{
		Action:  "invalid",
		Session: oapi.Work,
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}
	_, ok := resp.(oapi.CreateLogEntry400JSONResponse)
	if !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestCreateLogEntryInvalidSession(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: "invalid",
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}
	_, ok := resp.(oapi.CreateLogEntry400JSONResponse)
	if !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestCreateLogEntryNilPayload(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	body := oapi.CreateLogEntryRequest{
		Action:  oapi.Start,
		Session: oapi.Work,
	}

	resp, err := s.CreateLogEntry(context.Background(), oapi.CreateLogEntryRequestObject{
		Body: &body,
	})
	if err != nil {
		t.Fatalf("CreateLogEntry: %v", err)
	}

	createResp, ok := resp.(oapi.CreateLogEntry201JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if createResp.Payload != nil {
		t.Errorf("Payload = %v, want nil", createResp.Payload)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/svc/ -run TestCreateLogEntry -v`
Expected: FAIL — `undefined: Service.CreateLogEntry`

- [ ] **Step 3: Write minimal implementation**

Add to `internal/svc/service.go`:

```go
func (s *Service) CreateLogEntry(ctx context.Context, req oapi.CreateLogEntryRequestObject) (oapi.CreateLogEntryResponseObject, error) {
	if req.Body == nil {
		return oapi.CreateLogEntry400JSONResponse{Message: "missing request body"}, nil
	}

	action, err := xlate.LogActionToDomain(req.Body.Action)
	if err != nil {
		return oapi.CreateLogEntry400JSONResponse{
			Message: fmt.Sprintf("invalid action: %s", req.Body.Action),
		}, nil
	}

	session, err := xlate.SessionToDomain(req.Body.Session)
	if err != nil {
		return oapi.CreateLogEntry400JSONResponse{
			Message: fmt.Sprintf("invalid session: %s", req.Body.Session),
		}, nil
	}

	var payload map[string]any
	if req.Body.Payload != nil {
		payload = *req.Body.Payload
	}

	entry := domlog.Entry{
		Action:    action,
		Session:   session,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	id, err := s.logs.AddEntry(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("add log entry: %w", err)
	}

	respAction, err := xlate.LogActionFromDomain(action)
	if err != nil {
		return nil, fmt.Errorf("convert action: %w", err)
	}
	respSession, err := xlate.SessionFromDomain(session)
	if err != nil {
		return nil, fmt.Errorf("convert session: %w", err)
	}

	var respPayload *map[string]any
	if payload != nil {
		respPayload = &payload
	}

	return oapi.CreateLogEntry201JSONResponse{
		Id:        int64(id),
		Action:    respAction,
		Session:   respSession,
		Payload:   respPayload,
		Timestamp: entry.Timestamp,
	}, nil
}
```

Key decisions: svc sets `Timestamp = time.Now().UTC()` — svc is time-authority. [S5]. Action and session validated via `xlate.LogActionToDomain` and `xlate.SessionToDomain` — invalid values return 400. [S8]. `nil` payload stays `nil` — stored as `"null"` JSON by the port. [S6].

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/svc/ -run TestCreateLogEntry -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(svc): implement CreateLogEntry handler

Validates action/session enums, sets Timestamp to UTC
now, passes payload map to log.EventLog.AddEntry. [S5,
S6, S8]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 6: ListLogEntries handler

**Files:**
- Modify: `internal/svc/service.go`
- Test: `internal/svc/service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/svc/service_test.go`:

```go
func TestListLogEntries(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	for i := 0; i < 5; i++ {
		logs.AddEntry(context.Background(), domlog.Entry{
			Action:    domlog.ActionStart,
			Session:   domlog.SessionWork,
			Payload:   map[string]any{"index": i},
			Timestamp: time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC),
		})
	}

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if listResp.Entries == nil {
		t.Fatal("Entries is nil")
	}
	if len(*listResp.Entries) != 5 {
		t.Errorf("len(Entries) = %d, want 5", len(*listResp.Entries))
	}
	if listResp.Total == nil || *listResp.Total != 5 {
		t.Errorf("Total = %v, want 5", listResp.Total)
	}
}

func TestListLogEntriesDefaultLimitOffset(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	for i := 0; i < 60; i++ {
		logs.AddEntry(context.Background(), domlog.Entry{
			Action:    domlog.ActionStart,
			Session:   domlog.SessionWork,
			Timestamp: time.Now(),
		})
	}

	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 50 {
		t.Errorf("len(Entries) = %d, want 50 (default)", len(*listResp.Entries))
	}
	if *listResp.Total != 60 {
		t.Errorf("Total = %d, want 60", *listResp.Total)
	}
}

func TestListLogEntriesInvalidLimit(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog())

	badLimit := 0
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Limit: &badLimit},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(oapi.ListLogEntries400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for limit=0, got %T", resp)
	}
}

func TestListLogEntriesInvalidOffset(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog{})

	badOffset := -1
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Offset: &badOffset},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(oapi.ListLogEntries400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for offset=-1, got %T", resp)
	}
}

func TestListLogEntriesFilterByAction(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Now()})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionPause, Session: domlog.SessionWork, Timestamp: time.Now()})

	action := oapi.Start
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Action: &action},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*listResp.Entries))
	}
}

func TestListLogEntriesFilterBySession(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Now()})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionShortBreak, Timestamp: time.Now()})

	session := oapi.Work
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Session: &session},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}

	listResp, ok := resp.(oapi.ListLogEntries200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(*listResp.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(*listResp.Entries))
	}
}

func TestListLogEntriesFilterByTimeRange(t *testing.T) {
	logs := newRecordingEventLog()
	s := New(newRecordingNoteStore(), logs)

	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Date(2026, 1, 1, 0, 1, 30, 0, time.UTC)})
	logs.AddEntry(context.Background(), domlog.Entry{Action: domlog.ActionStart, Session: domlog.SessionWork, Timestamp: time.Date(2026, 1, 1, 0, 3, 0, 0, time.UTC)})

	from := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)

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
		t.Errorf("len(Entries) = %d, want 1", len(*listResp.Entries))
	}
}

func TestListLogEntriesInvalidAction(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog{})

	badAction := oapi.LogAction("invalid")
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Action: &badAction},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(oapi.ListLogEntries400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for invalid action, got %T", resp)
	}
}

func TestListLogEntriesInvalidSession(t *testing.T) {
	s := New(newRecordingNoteStore(), newRecordingEventLog{})

	badSession := oapi.PomodoroState("invalid")
	resp, err := s.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{
		Params: oapi.ListLogEntriesParams{Session: &badSession},
	})
	if err != nil {
		t.Fatalf("ListLogEntries: %v", err)
	}
	_, ok := resp.(oapi.ListLogEntries400JSONResponse)
	if !ok {
		t.Fatalf("expected 400 for invalid session, got %T", resp)
	}
}
```

Note: `ListLogEntries` in the OAPI spec also only defines a 200 response, no 400. Same as `ListNotes`, we need a custom 400 response type:

```go
type listLogEntries400Response struct {
	message string
}

func (r listLogEntries400Response) VisitListLogEntriesResponse(w nethttp.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	return json.NewEncoder(w).Encode(oapi.Error{Message: r.message})
}
```

Wait — the OAPI spec for `GET /logs` (listLogEntries) only defines a 200 response. But the generated code DOES have `CreateLogEntry400JSONResponse` (for `POST /logs`), not `ListLogEntries400JSONResponse`. Let me re-check. From `oapi_server.gen.go`:

- `CreateLogEntry400JSONResponse` exists (POST /logs has 400 in the spec).
- `ListLogEntries` only has `ListLogEntries200JSONResponse`.

So yes, we need a custom 400 type for `ListLogEntries`. Same as `ListNotes`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/svc/ -run TestListLogEntries -v`
Expected: FAIL — `undefined: Service.ListLogEntries`

- [ ] **Step 3: Write minimal implementation**

Add the 400 helper and `ListLogEntries` method to `internal/svc/service.go`:

```go
type listLogEntries400Response struct {
	message string
}

func (r listLogEntries400Response) VisitListLogEntriesResponse(w nethttp.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	return json.NewEncoder(w).Encode(oapi.Error{Message: r.message})
}

func (s *Service) ListLogEntries(ctx context.Context, req oapi.ListLogEntriesRequestObject) (oapi.ListLogEntriesResponseObject, error) {
	// Default limit=50, offset=0 [S15/S24]
	limit := 50
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	offset := 0
	if req.Params.Offset != nil {
		offset = *req.Params.Offset
	}

	// Validate [S15/S24]
	if limit < 1 || limit > 100 {
		return listLogEntries400Response{message: "limit must be between 1 and 100"}, nil
	}
	if offset < 0 {
		return listLogEntries400Response{message: "offset must be >= 0"}, nil
	}

	// Translate filters
	var actionFilter *domlog.Action
	if req.Params.Action != nil {
		action, err := xlate.LogActionToDomain(*req.Params.Action)
		if err != nil {
			return listLogEntries400Response{message: fmt.Sprintf("invalid action: %s", *req.Params.Action)}, nil
		}
		actionFilter = &action
	}

	var sessionFilter *domlog.Session
	if req.Params.Session != nil {
		session, err := xlate.SessionToDomain(*req.Params.Session)
		if err != nil {
			return listLogEntries400Response{message: fmt.Sprintf("invalid session: %s", *req.Params.Session)}, nil
		}
		sessionFilter = &session
	}

	// from/to: truncate to second, UTC [S5]
	var fromUTC, toUTC *time.Time
	if req.Params.From != nil {
		f := req.Params.From.UTC().Truncate(time.Second)
		fromUTC = &f
	}
	if req.Params.To != nil {
		// ceil to the whole second: if sub-second component exists, round up
		to := req.Params.To.UTC()
		if to.Nanosecond() > 0 {
			to = to.Add(time.Second - time.Duration(to.Nanosecond())*time.Nanosecond)
		}
		toUTC = &to
	}

	list, err := s.logs.ListEntries(ctx, adaptLog.ListParams{
		Action:  actionFilter,
		Session: sessionFilter,
		From:    fromUTC,
		To:      toUTC,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list log entries: %w", err)
	}

	entries := make([]oapi.LogEntry, 0, len(list.Entries))
	for _, e := range list.Entries {
		respAction, err := xlate.LogActionFromDomain(e.Action)
		if err != nil {
			return nil, fmt.Errorf("convert action: %w", err)
		}
		respSession, err := xlate.SessionFromDomain(e.Session)
		if err != nil {
			return nil, fmt.Errorf("convert session: %w", err)
		}

		var respPayload *map[string]any
		if e.Payload != nil {
			respPayload = &e.Payload
		}

		entries = append(entries, oapi.LogEntry{
			Id:        int64(e.ID),
			Action:    respAction,
			Session:   respSession,
			Payload:   respPayload,
			Timestamp: e.Timestamp,
		})
	}

	total := list.Total
	return oapi.ListLogEntries200JSONResponse{
		Entries: &entries,
		Total:   &total,
	}, nil
}
```

Key decisions: `from` truncates (floors) to the second; `to` ceils to the whole second if sub-second component exists. [S5]. All times converted to UTC. [S5]. Default limit=50, offset=0, validated same as ListNotes. [S15/S24]. Action and session filters translated via `xlate.LogActionToDomain` and `xlate.SessionToDomain`. Invalid values return 400.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/svc/ -v`
Expected: PASS — all tests pass

- [ ] **Step 5: Enable the compile-time check**

Uncomment the compile-time interface check in `service.go`:

```go
var _ oapi.StrictServerInterface = (*Service)(nil)
```

Run: `go build ./internal/svc/`
Expected: PASS — all six methods implemented

- [ ] **Step 6: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(svc): implement ListLogEntries handler

Filters by action, session, from/to time range with
default limit/offset. from floors to second, to ceils.
Enable compile-time StrictServerInterface check. [S5,
S15]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`
