# 03 — Service and storage debug call sites (issue 27) Sub-plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `complete-plan-execution` to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** `slog.Debug` diagnostics at the entry of every public svc handler and every sqlite store method, asserted by tests that also prove silence at the warn default (anti-leak, AC 6).

**Architecture:** Package-level `slog.Debug` calls at the top of each method body (before validation), one per method. Fields: operation name plus the identifying request parameters (id, limit/offset, filters). No signature churn anywhere.

**Tech Stack:** Go stdlib `log/slog`; tests capture output by `slog.SetDefault` with a `bytes.Buffer` handler and restore via `defer`.

Exact call sites (the enumeration agreed in the grill):

**`internal/svc/service.go`** — one line at the top of each of the 6 handlers:
`CreateNote`, `GetNote`, `DeleteNote`, `ListNotes`, `CreateLogEntry`, `ListLogEntries`.

**`internal/port/state/sqlite/note_store.go`** — one line at the top of each of the 4 methods:
`AddNote`, `ListNotes`, `GetNote`, `DeleteNote`.

**`internal/port/state/sqlite/event_log.go`** — one line at the top of each of the 2 methods:
`AddEntry`, `ListEntries`.

---

### Task 1: svc call sites

**Files:**
- Modify: `internal/svc/service.go`
- Test: `internal/svc/service_test.go` (append)

- [x] **Step 1: Write the failing test**

Append to `internal/svc/service_test.go` (reuse whatever constructor helpers the file already uses for `svc.New`; below uses the real stores via `sqlite.New(":memory:")` + `Migrate`, mirroring `internal/app/testdb_helper_test.go`):

```go
func captureSvcDebug(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	handler := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(handler)
	return &buf, func() { slog.SetDefault(prev) }
}

func TestSvcEmitsDebugDiagnostics(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	svcObj := svc.New(db.NoteStore(), db.EventLog())

	buf, restore := captureSvcDebug(t)
	defer restore()

	psBody := oapi.PomodoroState("work")
	_, _ = svcObj.CreateNote(context.Background(), oapi.CreateNoteRequestObject{
		Body: &oapi.CreateNoteJSONRequestBody{Note: "svc-log", PomodoroState: psBody},
	})
	_, _ = svcObj.ListNotes(context.Background(), oapi.ListNotesRequestObject{})
	_, _ = svcObj.ListLogEntries(context.Background(), oapi.ListLogEntriesRequestObject{})

	out := buf.String()
	for _, want := range []string{"svc: CreateNote", "svc: ListNotes", "svc: ListLogEntries"} {
		if !strings.Contains(out, want) {
			t.Errorf("debug diagnostics missing %q, got:\n%s", want, out)
		}
	}
}

func TestSvcSilentAtWarnDefault(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	svcObj := svc.New(db.NoteStore(), db.EventLog())

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(prev)

	psBody := oapi.PomodoroState("work")
	_, _ = svcObj.CreateNote(context.Background(), oapi.CreateNoteRequestObject{
		Body: &oapi.CreateNoteJSONRequestBody{Note: "quiet", PomodoroState: psBody},
	})

	if buf.Len() != 0 {
		t.Errorf("svc must be silent at warn default, got:\n%s", buf.String())
	}
}
```

Add the imports the file lacks: `"bytes"`, `"log/slog"`, `"strings"`, `"github.com/4sp1/vipo-go/internal/port/state/sqlite"` (the file already imports `oapi` and `context` under those aliases — reuse them exactly; no `pomodoro`/`xlate` needed since the tests pass string values).

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/svc/ -run 'TestSvc' -v`
Expected: FAIL — no debug diagnostic lines yet.

- [x] **Step 3: Write minimal implementation**

In `internal/svc/service.go`, add `"log/slog"` to imports and insert one line at the top of each handler (directly after the opening brace):

```go
func (s *Service) CreateNote(ctx context.Context, req oapi.CreateNoteRequestObject) (oapi.CreateNoteResponseObject, error) {
	slog.Debug("svc: CreateNote")

	// then the existing nil-body guard:
	if req.Body == nil {
```

To keep the nil-body guard honest, the line goes first and reads the safe fields only. Full set:

```go
// CreateNote, after the opening brace:
slog.Debug("svc: CreateNote")

// GetNote:
slog.Debug("svc: GetNote", "id", req.Id)

// DeleteNote:
slog.Debug("svc: DeleteNote", "id", req.Id)

// ListNotes:
slog.Debug("svc: ListNotes", "limit", req.Params.Limit, "offset", req.Params.Offset)

// CreateLogEntry:
slog.Debug("svc: CreateLogEntry")

// ListLogEntries:
slog.Debug("svc: ListLogEntries", "limit", req.Params.Limit, "offset", req.Params.Offset, "action", req.Params.Action, "session", req.Params.Session)
```

Rationale for the guards: `CreateNote`/`CreateLogEntry` read `req.Body` fields after a nil check, so the entry line logs nothing about the body (bodies belong to the HTTP middleware capture); list handlers dereference `req.Params` freely because generated defaults populate it.

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/svc/ -v && go build ./...`
Expected: PASS.

- [x] **Step 5: Commit with caveman-commit**

Use `caveman-commit` for the message content, then the commit sequence from `core-commands` (mktemp → Read temp file → Write message → `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`).

Message sketch: `feat(svc): debug diagnostics per handler` (subject ≤50 chars).

---

### Task 2: storage call sites

**Files:**
- Modify: `internal/port/state/sqlite/note_store.go`
- Modify: `internal/port/state/sqlite/event_log.go`
- Test: `internal/port/state/sqlite/note_store_test.go` (append)

- [x] **Step 1: Write the failing test**

Append to `internal/port/state/sqlite/note_store_test.go` (the file already has `setupTestDB`):

```go
func TestStorageEmitsDebugDiagnostics(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	n := domnote.State{
		Note:      "storage-log",
		Pomodoro:  pomodoro.Work,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	_, _ = ns.AddNote(context.Background(), n)
	_, _ = ns.GetNote(context.Background(), 1)
	_ = ns.DeleteNote(context.Background(), 1)
	_, _ = ns.ListNotes(context.Background(), adaptNote.ListParams{Limit: 10, Offset: 0})

	out := buf.String()
	for _, want := range []string{"sqlite: AddNote", "sqlite: GetNote", "sqlite: DeleteNote", "sqlite: ListNotes"} {
		if !strings.Contains(out, want) {
			t.Errorf("storage diagnostics missing %q, got:\n%s", want, out)
		}
	}
}
```

Add imports the file lacks: `"bytes"`, `"log/slog"`, `"strings"` (existing file already imports `context`, `time`, `adaptNote`, `domnote`, `pomodoro`).

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/port/state/sqlite/ -run 'TestStorage' -v`
Expected: FAIL — no debug lines yet.

- [x] **Step 3: Write minimal implementation**

Add `"log/slog"` to the imports of both files. Insert one line at the top of each method:

`internal/port/state/sqlite/note_store.go`:

```go
func (s *noteStore) AddNote(ctx context.Context, n domnote.State) (domnote.ID, error) {
	slog.Debug("sqlite: AddNote", "pomodoro_state", int(n.Pomodoro), "created_at", n.CreatedAt.Unix())

func (s *noteStore) ListNotes(ctx context.Context, params adaptNote.ListParams) (adaptNote.List, error) {
	slog.Debug("sqlite: ListNotes", "limit", params.Limit, "offset", params.Offset, "has_filter", params.PomodoroState != nil)

func (s *noteStore) GetNote(ctx context.Context, id domnote.ID) (domnote.State, error) {
	slog.Debug("sqlite: GetNote", "id", int64(id))

func (s *noteStore) DeleteNote(ctx context.Context, id domnote.ID) error {
	slog.Debug("sqlite: DeleteNote", "id", int64(id))
```

`internal/port/state/sqlite/event_log.go`:

```go
func (el *eventLog) AddEntry(ctx context.Context, entry domlog.Entry) (domlog.ID, error) {
	slog.Debug("sqlite: AddEntry", "action", int(entry.Action), "session", int(entry.Session), "timestamp", entry.Timestamp.Unix())

func (el *eventLog) ListEntries(ctx context.Context, params adaptLog.ListParams) (adaptLog.EntryList, error) {
	slog.Debug("sqlite: ListEntries", "limit", params.Limit, "offset", params.Offset, "has_action", params.Action != nil, "has_session", params.Session != nil)
```

(Each line inserted as the first statement, after the opening brace; the shown signature line contextualizes placement, not a duplicate.)

- [x] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/port/state/sqlite/ -v && go test ./... -count=1 && go vet ./...`
Expected: PASS everywhere.

- [x] **Step 5: Commit with caveman-commit**

Use `caveman-commit` for the message content, then the commit sequence from `core-commands` (mktemp → Read temp file → Write message → `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`).

Message sketch: `feat(sqlite): debug diagnostics per store method` (subject ≤50 chars).
