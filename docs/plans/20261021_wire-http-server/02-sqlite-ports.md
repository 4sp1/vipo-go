# 02 — SQLite Ports

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the SQLite port as a shared `*DB` wrapper exposing `NoteStore()` and `EventLog()` over one `*sql.DB`, with storage-boundary enum converters and full implementations of `note.Store` + `log.EventLog`.

**Architecture:** Single `port/state/sqlite` package. `New(file string) (*DB, error)` opens the DB and returns a `*DB` struct holding the shared `*sql.DB`. Two sub-structs — `noteStore` and `eventLog` — implement the adapter interfaces. Unexported `*FromStorage` converters scan raw `int64` from the DB and validate before returning domain enums. [S4], [S21], [S12], [S10], [S6], [S7], [S5], [S3], [S20].

**Tech Stack:** Go stdlib (`context`, `database/sql`, `encoding/json`, `errors`, `fmt`, `time`), `glebarez/go-sqlite` driver, `adapter/state/note`, `adapter/state/log`, domain packages.

**Depends on:** Sub-plan 01 (adapter interfaces `note.Store`, `log.EventLog`, typed `ErrNotFound`, updated `log.Entry` with `ID` + `map[string]any` payload).

---

## Files

- Create: `internal/port/state/sqlite/converters.go`
- Create: `internal/port/state/sqlite/db.go`
- Create: `internal/port/state/sqlite/note_store.go`
- Create: `internal/port/state/sqlite/event_log.go`
- Delete: `internal/port/state/sqlite/port.go`

---

### Task 1: Storage-boundary enum converters

**Files:**
- Create: `internal/port/state/sqlite/converters.go`
- Test: `internal/port/state/sqlite/converters_test.go`

- [ ] **Step 1: Write the failing test**

```go
package sqlite

import (
	"testing"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func TestPomodoroStateFromStorage(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    pomodoro.State
		wantErr bool
	}{
		{"work", 0, pomodoro.Work, false},
		{"short_break", 1, pomodoro.ShortBreak, false},
		{"long_break", 2, pomodoro.LongBreak, false},
		{"negative", -1, pomodoro.State(0), true},
		{"out_of_range", 3, pomodoro.State(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PomodoroStateFromStorage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLogActionFromStorage(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    log.Action
		wantErr bool
	}{
		{"unknown", 0, log.Action(0), true},
		{"start", 1, log.ActionStart, false},
		{"pause", 2, log.ActionPause, false},
		{"reset", 3, log.ActionReset, false},
		{"expire", 4, log.ActionExpire, false},
		{"resume", 5, log.ActionResume, false},
		{"select", 6, log.ActionSelect, false},
		{"new_note_removed", 7, log.Action(0), true},
		{"negative", -1, log.Action(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LogActionFromStorage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLogSessionFromStorage(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    log.Session
		wantErr bool
	}{
		{"unknown", 0, log.Session(0), true},
		{"work", 1, log.SessionWork, false},
		{"short_break", 2, log.SessionShortBreak, false},
		{"long_break", 3, log.SessionLongBreak, false},
		{"out_of_range", 4, log.Session(0), true},
		{"negative", -1, log.Session(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LogSessionFromStorage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/port/state/sqlite/ -run TestPomodoroStateFromStorage -v`
Expected: FAIL — `undefined: PomodoroStateFromStorage`

- [ ] **Step 3: Write minimal implementation**

Create `internal/port/state/sqlite/converters.go`:

```go
package sqlite

import (
	"fmt"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func PomodoroStateFromStorage(v int64) (pomodoro.State, error) {
	switch v {
	case 0:
		return pomodoro.Work, nil
	case 1:
		return pomodoro.ShortBreak, nil
	case 2:
		return pomodoro.LongBreak, nil
	default:
		return pomodoro.State(0), fmt.Errorf("invalid pomodoro state from storage: %d", v)
	}
}

func LogActionFromStorage(v int64) (log.Action, error) {
	switch v {
	case log.ActionStart:
		return log.ActionStart, nil
	case log.ActionPause:
		return log.ActionPause, nil
	case log.ActionReset:
		return log.ActionReset, nil
	case log.ActionExpire:
		return log.ActionExpire, nil
	case log.ActionResume:
		return log.ActionResume, nil
	case log.ActionSelect:
		return log.ActionSelect, nil
	default:
		return log.Action(0), fmt.Errorf("invalid log action from storage: %d", v)
	}
}

func LogSessionFromStorage(v int64) (log.Session, error) {
	switch v {
	case log.SessionWork:
		return log.SessionWork, nil
	case log.SessionShortBreak:
		return log.SessionShortBreak, nil
	case log.SessionLongBreak:
		return log.SessionLongBreak, nil
	default:
		return log.Session(0), fmt.Errorf("invalid log session from storage: %d", v)
	}
}
```

Note: `LogActionFromStorage` rejects `0` (`ActionUnknown`) and `7` (the removed `ActionNewNote` value) via the `default` case. [S20], [S4].

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/port/state/sqlite/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(port): add storage-boundary enum converters

PomodoroStateFromStorage, LogActionFromStorage,
LogSessionFromStorage scan raw int64 from DB and
validate before returning domain enums. Unknown and
removed values (ActionUnknown, ActionNewNote) are
rejected. [S4]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 2: Shared `*DB` wrapper

**Files:**
- Create: `internal/port/state/sqlite/db.go`
- Test: `internal/port/state/sqlite/db_test.go`

- [ ] **Step 1: Write the failing test**

```go
package sqlite

import (
	"context"
	"testing"
)

func TestNewOpensDatabase(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestNoteStoreReturnsImplementor(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	ns := db.NoteStore()
	if ns == nil {
		t.Error("NoteStore returned nil")
	}
}

func TestEventLogReturnsImplementor(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	el := db.EventLog()
	if el == nil {
		t.Error("EventLog returned nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/port/state/sqlite/ -run TestNewOpensDatabase -v`
Expected: FAIL — `undefined: New`, `undefined: DB`

- [ ] **Step 3: Write minimal implementation**

Create `internal/port/state/sqlite/db.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

type DB struct {
	db *sql.DB
}

func New(file string) (*DB, error) {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) NoteStore() *noteStore {
	return &noteStore{db: d.db}
}

func (d *DB) EventLog() *eventLog {
	return &eventLog{db: d.db}
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) PingContext(ctx context.Context) error {
	return d.db.PingContext(ctx)
}
```

Note: `NoteStore()` and `EventLog()` return concrete struct pointers, which satisfy the `note.Store` and `log.EventLog` interfaces respectively (implemented in Tasks 3 and 4). [S21].

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/port/state/sqlite/ -run TestNew -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
refactor(port): replace main() with shared *DB wrapper

New(file) returns *DB exposing NoteStore() and EventLog()
over one shared *sql.DB. Replaces the func main(file) wart.
[S21]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 3: Note SQLite port

**Files:**
- Create: `internal/port/state/sqlite/note_store.go`
- Delete: `internal/port/state/sqlite/port.go`
- Test: `internal/port/state/sqlite/note_store_test.go`

This task requires a test database with the `notes` table. Since migrations are handled in sub-plan 03, the test creates the table manually via `db.Exec`.

- [ ] **Step 1: Write the failing test**

```go
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = db.db.Exec(`CREATE TABLE notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		note TEXT NOT NULL,
		pomodoro_state INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNoteStoreAddNoteReturnsID(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	n := domnote.State{
		Note:      "test note",
		Pomodoro:  pomodoro.Work,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}

	id2, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote 2: %v", err)
	}
	if id2 != 2 {
		t.Errorf("id = %d, want 2", id2)
	}
}

func TestNoteStoreGetNote(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	n := domnote.State{
		Note:      "hello",
		Pomodoro:  pomodoro.ShortBreak,
		CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	id, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	got, err := ns.GetNote(context.Background(), id)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if got.Note != "hello" {
		t.Errorf("Note = %q, want hello", got.Note)
	}
	if got.Pomodoro != pomodoro.ShortBreak {
		t.Errorf("Pomodoro = %d, want %d", got.Pomodoro, pomodoro.ShortBreak)
	}
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}
	wantTS := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if !got.CreatedAt.Equal(wantTS) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, wantTS)
	}
}

func TestNoteStoreGetNoteNotFound(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	_, err := ns.GetNote(context.Background(), 999)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("err = %v, want adaptNote.ErrNotFound", err)
	}
}

func TestNoteStoreDeleteNote(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	n := domnote.State{
		Note:     "to delete",
		Pomodoro: pomodoro.Work,
	}
	id, err := ns.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}

	if err := ns.DeleteNote(context.Background(), id); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	_, err = ns.GetNote(context.Background(), id)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("GetNote after delete: err = %v, want ErrNotFound", err)
	}
}

func TestNoteStoreDeleteNoteNotFound(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	err := ns.DeleteNote(context.Background(), 999)
	if !errors.Is(err, adaptNote.ErrNotFound) {
		t.Errorf("err = %v, want adaptNote.ErrNotFound", err)
	}
}

func TestNoteStoreListNotes(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	for i := 0; i < 5; i++ {
		n := domnote.State{
			Note:      "note",
			Pomodoro:  pomodoro.Work,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC),
		}
		_, err := ns.AddNote(context.Background(), n)
		if err != nil {
			t.Fatalf("AddNote %d: %v", i, err)
		}
	}

	list, err := ns.ListNotes(context.Background(), adaptNote.ListParams{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(list.Notes) != 3 {
		t.Errorf("len(Notes) = %d, want 3", len(list.Notes))
	}
	if list.Total != 5 {
		t.Errorf("Total = %d, want 5", list.Total)
	}
}

func TestNoteStoreListNotesByPomodoroState(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	work := pomodoro.Work
	breakState := pomodoro.ShortBreak

	for i := 0; i < 3; i++ {
		_, err := ns.AddNote(context.Background(), domnote.State{Note: "work", Pomodoro: work, CreatedAt: time.Now()})
		if err != nil {
			t.Fatalf("AddNote: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := ns.AddNote(context.Background(), domnote.State{Note: "break", Pomodoro: breakState, CreatedAt: time.Now()})
		if err != nil {
			t.Fatalf("AddNote: %v", err)
		}
	}

	list, err := ns.ListNotes(context.Background(), adaptNote.ListParams{PomodoroState: &breakState, Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(list.Notes) != 2 {
		t.Errorf("len(Notes) = %d, want 2", len(list.Notes))
	}
	if list.Total != 2 {
		t.Errorf("Total = %d, want 2", list.Total)
	}
}

func TestNoteStoreListNotesEmpty(t *testing.T) {
	db := setupTestDB(t)
	ns := db.NoteStore()

	list, err := ns.ListNotes(context.Background(), adaptNote.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if list.Notes == nil {
		t.Error("Notes is nil, want empty slice")
	}
	if len(list.Notes) != 0 {
		t.Errorf("len(Notes) = %d, want 0", len(list.Notes))
	}
	if list.Total != 0 {
		t.Errorf("Total = %d, want 0", list.Total)
	}
}

var _ = sql.ErrNoRows
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/port/state/sqlite/ -run TestNoteStore -v`
Expected: FAIL — `undefined: noteStore`, `undefined: db.db` (field access in test)

- [ ] **Step 3: Write minimal implementation**

First, delete the old port file:

Run: `rm internal/port/state/sqlite/port.go`

Create `internal/port/state/sqlite/note_store.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
)

type noteStore struct {
	db *sql.DB
}

func (s *noteStore) AddNote(ctx context.Context, n domnote.State) (domnote.ID, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO notes (note, pomodoro_state, created_at) VALUES (?, ?, ?)",
		n.Note, int(n.Pomodoro), n.CreatedAt.Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert note: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return domnote.ID(id), nil
}

func (s *noteStore) ListNotes(ctx context.Context, params adaptNote.ListParams) (adaptNote.List, error) {
	whereClause := ""
	args := []any{}

	if params.PomodoroState != nil {
		whereClause = " WHERE pomodoro_state = ?"
		args = append(args, int(*params.PomodoroState))
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM notes" + whereClause
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return adaptNote.List{}, fmt.Errorf("count notes: %w", err)
	}

	query := "SELECT id, note, pomodoro_state, created_at FROM notes" + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return adaptNote.List{}, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	notes := make([]domnote.State, 0)
	for rows.Next() {
		var n domnote.State
		var pomodoroRaw, createdAtUnix int64
		if err := rows.Scan(&n.ID, &n.Note, &pomodoroRaw, &createdAtUnix); err != nil {
			return adaptNote.List{}, fmt.Errorf("scan note: %w", err)
		}
		ps, err := PomodoroStateFromStorage(pomodoroRaw)
		if err != nil {
			return adaptNote.List{}, fmt.Errorf("convert pomodoro state: %w", err)
		}
		n.Pomodoro = ps
		n.CreatedAt = time.Unix(createdAtUnix, 0)
		notes = append(notes, n)
	}

	if err := rows.Err(); err != nil {
		return adaptNote.List{}, fmt.Errorf("rows: %w", err)
	}

	return adaptNote.List{
		Notes: notes,
		Total: total,
	}, nil
}

func (s *noteStore) GetNote(ctx context.Context, id domnote.ID) (domnote.State, error) {
	var n domnote.State
	var pomodoroRaw, createdAtUnix int64

	err := s.db.QueryRowContext(ctx,
		"SELECT id, note, pomodoro_state, created_at FROM notes WHERE id = ?",
		id,
	).Scan(&n.ID, &n.Note, &pomodoroRaw, &createdAtUnix)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domnote.State{}, adaptNote.ErrNotFound
		}
		return domnote.State{}, fmt.Errorf("get note: %w", err)
	}

	ps, err := PomodoroStateFromStorage(pomodoroRaw)
	if err != nil {
		return domnote.State{}, fmt.Errorf("convert pomodoro state: %w", err)
	}
	n.Pomodoro = ps
	n.CreatedAt = time.Unix(createdAtUnix, 0)

	return n, nil
}

func (s *noteStore) DeleteNote(ctx context.Context, id domnote.ID) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM notes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if affected == 0 {
		return adaptNote.ErrNotFound
	}

	return nil
}
```

Key changes from the old `port.go`: `AddNote` returns `(domnote.ID, error)` via `LastInsertId`; `ListNotes`/`GetNote` scan `pomodoro_state` into `int64` then validate via `PomodoroStateFromStorage` instead of scanning directly into the domain enum; not-found returns `adaptNote.ErrNotFound` instead of `fmt.Errorf("note not found: %d", id)`. [S4], [S12], [S3].

The test references `db.db` (the unexported field) for table setup. Since the test is in the same package, this works. An alternative is adding an unexported `exec` helper on `*DB`; for simplicity the direct field access is used within the package.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/port/state/sqlite/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(port): implement note SQLite port with converters

AddNote returns autoincremented ID via LastInsertId.
ListNotes/GetNote scan pomodoro_state into int64 then
validate via PomodoroStateFromStorage. Not-found returns
typed adaptNote.ErrNotFound. [S4, S12, S3]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 4: Log SQLite port

**Files:**
- Create: `internal/port/state/sqlite/event_log.go`
- Test: `internal/port/state/sqlite/event_log_test.go`

- [ ] **Step 1: Write the failing test**

```go
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

func setupLogTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = db.db.Exec(`CREATE TABLE log_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		action INTEGER NOT NULL,
		session INTEGER NOT NULL,
		payload TEXT NOT NULL,
		timestamp INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestEventLogAddEntryReturnsID(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	entry := domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   map[string]any{"key": "value"},
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, err := el.AddEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}
}

func TestEventLogAddEntryNilPayload(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	entry := domlog.Entry{
		Action:    domlog.ActionPause,
		Session:   domlog.SessionShortBreak,
		Payload:   nil,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	_, err := el.AddEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("Total = %d, want 1", list.Total)
	}
	if list.Entries[0].Payload != nil {
		t.Errorf("Payload = %v, want nil", list.Entries[0].Payload)
	}
}

func TestEventLogListEntries(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	for i := 0; i < 5; i++ {
		entry := domlog.Entry{
			Action:    domlog.ActionStart,
			Session:   domlog.SessionWork,
			Payload:   map[string]any{"index": i},
			Timestamp: time.Date(2026, 1, 1, 0, i, 0, 0, time.UTC),
		}
		_, err := el.AddEntry(context.Background(), entry)
		if err != nil {
			t.Fatalf("AddEntry %d: %v", i, err)
		}
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(list.Entries))
	}
	if list.Total != 5 {
		t.Errorf("Total = %d, want 5", list.Total)
	}
}

func TestEventLogListEntriesFilterByAction(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	entries := []struct {
		action  domlog.Action
		session domlog.Session
	}{
		{domlog.ActionStart, domlog.SessionWork},
		{domlog.ActionPause, domlog.SessionWork},
		{domlog.ActionStart, domlog.SessionShortBreak},
	}
	for _, e := range entries {
		_, err := el.AddEntry(context.Background(), domlog.Entry{
			Action:    e.action,
			Session:   e.session,
			Payload:   map[string]any{},
			Timestamp: time.Now(),
		})
		if err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	}

	action := domlog.ActionStart
	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Action: &action, Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 2 {
		t.Errorf("len(Entries) = %d, want 2", len(list.Entries))
	}
	if list.Total != 2 {
		t.Errorf("Total = %d, want 2", list.Total)
	}
}

func TestEventLogListEntriesFilterBySession(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	session := domlog.SessionWork
	_, err := el.AddEntry(context.Background(), domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   map[string]any{},
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionShortBreak,
		Payload:   map[string]any{},
		Timestamp: time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Session: &session, Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(list.Entries))
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}

func TestEventLogListEntriesFilterByTimeRange(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	from := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)

	_, err := el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 0, 1, 30, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 0, 3, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{From: &from, To: &to, Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(list.Entries))
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}

func TestEventLogListEntriesEmpty(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if list.Entries == nil {
		t.Error("Entries is nil, want empty slice")
	}
	if len(list.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(list.Entries))
	}
	if list.Total != 0 {
		t.Errorf("Total = %d, want 0", list.Total)
	}
}

func TestEventLogPayloadRoundTrip(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	original := map[string]any{
		"string": "hello",
		"number": float64(42),
		"nested": map[string]any{"inner": "value"},
		"array":  []any{1, 2, 3},
	}
	entry := domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   original,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	id, err := el.AddEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}

	got := list.Entries[0]
	if got.ID != id {
		t.Errorf("ID = %d, want %d", got.ID, id)
	}

	originalJSON, _ := json.Marshal(original)
	gotJSON, _ := json.Marshal(got.Payload)
	if string(originalJSON) != string(gotJSON) {
		t.Errorf("payload mismatch:\n  orig: %s\n  got:  %s", originalJSON, gotJSON)
	}
}

func TestEventLogListEntriesFiltersCombined(t *testing.T) {
	db := setupLogTestDB(t)
	el := db.EventLog()

	action := domlog.ActionPause
	session := domlog.SessionWork
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 23, 59, 59, 0, time.UTC)

	_, err := el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionPause, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionStart, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionPause, Session: domlog.SessionShortBreak,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	_, err = el.AddEntry(context.Background(), domlog.Entry{
		Action: domlog.ActionPause, Session: domlog.SessionWork,
		Payload: map[string]any{}, Timestamp: time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	list, err := el.ListEntries(context.Background(), adaptLog.ListParams{
		Action:  &action,
		Session: &session,
		From:    &from,
		To:      &to,
		Limit:   50,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(list.Entries) != 1 {
		t.Errorf("len(Entries) = %d, want 1", len(list.Entries))
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/port/state/sqlite/ -run TestEventLog -v`
Expected: FAIL — `undefined: eventLog`, undefined methods on `*DB.EventLog()`

- [ ] **Step 3: Write minimal implementation**

Create `internal/port/state/sqlite/event_log.go`:

```go
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

type eventLog struct {
	db *sql.DB
}

func (el *eventLog) AddEntry(ctx context.Context, entry domlog.Entry) (domlog.ID, error) {
	var payloadBytes []byte
	var err error
	if entry.Payload != nil {
		payloadBytes, err = json.Marshal(entry.Payload)
		if err != nil {
			return 0, fmt.Errorf("marshal payload: %w", err)
		}
	} else {
		payloadBytes = []byte("null")
	}

	result, err := el.db.ExecContext(ctx,
		"INSERT INTO log_entries (action, session, payload, timestamp) VALUES (?, ?, ?, ?)",
		int(entry.Action), int(entry.Session), string(payloadBytes), entry.Timestamp.Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert log entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return domlog.ID(id), nil
}

func (el *eventLog) ListEntries(ctx context.Context, params adaptLog.ListParams) (adaptLog.EntryList, error) {
	var conditions []string
	var args []any

	if params.Action != nil {
		conditions = append(conditions, "action = ?")
		args = append(args, int(*params.Action))
	}
	if params.Session != nil {
		conditions = append(conditions, "session = ?")
		args = append(args, int(*params.Session))
	}
	if params.From != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, params.From.Unix())
	}
	if params.To != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, params.To.Unix())
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM log_entries" + whereClause
	if err := el.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return adaptLog.EntryList{}, fmt.Errorf("count log entries: %w", err)
	}

	query := "SELECT id, action, session, payload, timestamp FROM log_entries" + whereClause + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := el.db.QueryContext(ctx, query, args...)
	if err != nil {
		return adaptLog.EntryList{}, fmt.Errorf("query log entries: %w", err)
	}
	defer rows.Close()

	entries := make([]domlog.Entry, 0)
	for rows.Next() {
		var e domlog.Entry
		var actionRaw, sessionRaw, timestampUnix int64
		var payloadStr string

		if err := rows.Scan(&e.ID, &actionRaw, &sessionRaw, &payloadStr, &timestampUnix); err != nil {
			return adaptLog.EntryList{}, fmt.Errorf("scan log entry: %w", err)
		}

		action, err := LogActionFromStorage(actionRaw)
		if err != nil {
			return adaptLog.EntryList{}, fmt.Errorf("convert action: %w", err)
		}
		session, err := LogSessionFromStorage(sessionRaw)
		if err != nil {
			return adaptLog.EntryList{}, fmt.Errorf("convert session: %w", err)
		}
		e.Action = action
		e.Session = session
		e.Timestamp = time.Unix(timestampUnix, 0)

		if payloadStr != "null" && payloadStr != "" {
			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				return adaptLog.EntryList{}, fmt.Errorf("unmarshal payload: %w", err)
			}
			e.Payload = payload
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return adaptLog.EntryList{}, fmt.Errorf("rows: %w", err)
	}

	return adaptLog.EntryList{
		Entries: entries,
		Total:   total,
	}, nil
}
```

Key decisions: `payload` is `json.Marshal`'d on write and `json.Unmarshal`'d into `map[string]any` on read — semantically lossless, not byte-faithful. [S6]. `nil` payload stored as `"null"` JSON and deserialized back to `nil`. Action and session scanned into `int64` then validated via `*FromStorage`. [S4]. `total` via `COUNT(*)` from the same `WHERE` clause. [S7].

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/port/state/sqlite/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required though empty)
3. Write the commit message to the temp file:

```
feat(port): implement log SQLite port

AddEntry marshals payload to JSON, returns autoincremented
ID. ListEntries filters by action, session, from/to time
range with COUNT(*) total. Reads validate via
LogActionFromStorage/LogSessionFromStorage. [S4, S6, S7,
S10]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 5: Delete old `port.go` and verify build

**Files:**
- Delete: `internal/port/state/sqlite/port.go` (if still present)

- [ ] **Step 1: Verify port.go is deleted**

If `port.go` was not deleted in Task 3, delete it now:

Run: `rm internal/port/state/sqlite/port.go`

- [ ] **Step 2: Verify the build passes**

Run: `go build ./internal/port/state/sqlite/...`
Expected: PASS

Run: `go build ./internal/port/state/sqlite/ ./internal/adapter/state/note/ ./internal/adapter/state/log/`
Expected: PASS — all new packages compile. The full `go build ./...` will still fail if `internal/svc/` and `internal/app/` reference old types, but those directories are empty.

- [ ] **Step 3: Run all tests**

Run: `go test ./internal/port/state/sqlite/ -v`
Expected: PASS

- [ ] **Step 4: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
chore(port): remove old port.go, verify sqlite build

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`
