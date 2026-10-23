# 01 — Domain & Adapter Restructuring

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the `log.Entry` domain model (add ID, change payload to `map[string]any`, delete dead code) and split `adapter/state` into `note.Store` + `log.EventLog` with typed `ErrNotFound`.

**Architecture:** Two adapter packages mirroring the domain split: `adapter/state/note` (`package note`) and `adapter/state/log` (`package log`). Each defines a port interface using domain types and a typed `ErrNotFound`. The old `adapter/state/repository.go` is deleted.

**Tech Stack:** Go stdlib only (`context`, `errors`, `time`).

---

## Files

- Modify: `internal/domain/state/log/entry.go`
- Create: `internal/adapter/state/note/store.go`
- Create: `internal/adapter/state/log/eventlog.go`
- Delete: `internal/adapter/state/repository.go`

---

### Task 1: Update `log.Entry` domain model

**Files:**
- Modify: `internal/domain/state/log/entry.go`
- Test: `internal/domain/state/log/entry_test.go`

- [ ] **Step 1: Write the failing test**

```go
package log

import (
	"testing"
	"time"
)

func TestEntryConstruction(t *testing.T) {
	entry := Entry{
		ID:        ID(42),
		Action:    ActionStart,
		Session:   SessionWork,
		Payload:   map[string]any{"key": "value"},
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if entry.ID != 42 {
		t.Errorf("ID = %d, want 42", entry.ID)
	}
	if entry.Action != ActionStart {
		t.Errorf("Action = %d, want %d", entry.Action, ActionStart)
	}
	if entry.Session != SessionWork {
		t.Errorf("Session = %d, want %d", entry.Session, SessionWork)
	}
	if entry.Payload["key"] != "value" {
		t.Errorf("Payload[key] = %v, want value", entry.Payload["key"])
	}
}

func TestActionValues(t *testing.T) {
	tests := []struct {
		name  string
		value Action
		want  int
	}{
		{"ActionUnknown", ActionUnknown, 0},
		{"ActionStart", ActionStart, 1},
		{"ActionPause", ActionPause, 2},
		{"ActionReset", ActionReset, 3},
		{"ActionExpire", ActionExpire, 4},
		{"ActionResume", ActionResume, 5},
		{"ActionSelect", ActionSelect, 6},
	}
	for _, tt := range tests {
		if int(tt.value) != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}

func TestSessionValues(t *testing.T) {
	tests := []struct {
		name  string
		value Session
		want  int
	}{
		{"SessionUnknown", SessionUnknown, 0},
		{"SessionWork", SessionWork, 1},
		{"SessionShortBreak", SessionShortBreak, 2},
		{"SessionLongBreak", SessionLongBreak, 3},
	}
	for _, tt := range tests {
		if int(tt.value) != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/state/log/ -run TestEntryConstruction -v`
Expected: FAIL — `undefined: ID` or `cannot use map[string]any{...} as type Message`

- [ ] **Step 3: Write minimal implementation**

Replace the entire contents of `internal/domain/state/log/entry.go` with:

```go
package log

import "time"

type ID int64

type Entry struct {
	ID        ID
	Action    Action
	Session   Session
	Payload   map[string]any
	Timestamp time.Time
}

type Action int

const (
	ActionUnknown Action = iota
	ActionStart
	ActionPause
	ActionReset
	ActionExpire
	ActionResume
	ActionSelect
)

type Session int

const (
	SessionUnknown Session = iota
	SessionWork
	SessionShortBreak
	SessionLongBreak
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/state/log/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

Use `caveman-commit` for the message content, then the commit sequence from `core-commands`:

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
refactor(domain): add ID to log.Entry, payload to map

Remove Message sealed interface and MessageNewNote (dead
code — new_note excluded from log endpoints). Change Payload
to map[string]any to match free-form OAPI contract. Remove
ActionNewNote constant.

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 2: Create `adapter/state/note/store.go`

**Files:**
- Create: `internal/adapter/state/note/store.go`
- Test: `internal/adapter/state/note/store_test.go`

- [ ] **Step 1: Write the failing test**

```go
package note

import (
	"context"
	"testing"

	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

type mockStore struct {
	notes map[domnote.ID]domnote.State
}

func (m *mockStore) AddNote(_ context.Context, n domnote.State) (domnote.ID, error) {
	id := domnote.ID(len(m.notes) + 1)
	m.notes[id] = n
	return id, nil
}

func (m *mockStore) ListNotes(_ context.Context, _ ListParams) (List, error) {
	notes := make([]domnote.State, 0, len(m.notes))
	for _, n := range m.notes {
		notes = append(notes, n)
	}
	return List{Notes: notes, Total: len(notes)}, nil
}

func (m *mockStore) GetNote(_ context.Context, id domnote.ID) (domnote.State, error) {
	n, ok := m.notes[id]
	if !ok {
		return domnote.State{}, ErrNotFound
	}
	return n, nil
}

func (m *mockStore) DeleteNote(_ context.Context, id domnote.ID) error {
	delete(m.notes, id)
	return nil
}

func TestStoreInterface(t *testing.T) {
	var s Store = &mockStore{notes: make(map[domnote.ID]domnote.State)}
	n := domnote.State{Note: "test", Pomodoro: pomodoro.Work}
	id, err := s.AddNote(context.Background(), n)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}
	got, err := s.GetNote(context.Background(), id)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if got.Note != "test" {
		t.Errorf("Note = %q, want test", got.Note)
	}
}

func TestErrNotFound(t *testing.T) {
	s := &mockStore{notes: make(map[domnote.ID]domnote.State)}
	_, err := s.GetNote(context.Background(), 999)
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/state/note/ -v`
Expected: FAIL — `undefined: Store`, `undefined: ListParams`, `undefined: List`, `undefined: ErrNotFound`

- [ ] **Step 3: Write minimal implementation**

Create `internal/adapter/state/note/store.go`:

```go
package note

import (
	"context"
	"errors"

	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

type ListParams struct {
	PomodoroState *pomodoro.State
	Limit, Offset int
}

type List struct {
	Notes []domnote.State
	Total int
}

type Store interface {
	AddNote(ctx context.Context, n domnote.State) (domnote.ID, error)
	ListNotes(ctx context.Context, params ListParams) (List, error)
	GetNote(ctx context.Context, id domnote.ID) (domnote.State, error)
	DeleteNote(ctx context.Context, id domnote.ID) error
}

var ErrNotFound = errors.New("adapter: note not found")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/state/note/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(adapter): add note.Store port interface

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 3: Create `adapter/state/log/eventlog.go`

**Files:**
- Create: `internal/adapter/state/log/eventlog.go`
- Test: `internal/adapter/state/log/eventlog_test.go`

- [ ] **Step 1: Write the failing test**

```go
package log

import (
	"context"
	"testing"
	"time"

	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

type mockEventLog struct {
	entries map[domlog.ID]domlog.Entry
}

func (m *mockEventLog) AddEntry(_ context.Context, e domlog.Entry) (domlog.ID, error) {
	id := domlog.ID(len(m.entries) + 1)
	m.entries[id] = e
	return id, nil
}

func (m *mockEventLog) ListEntries(_ context.Context, _ ListParams) (EntryList, error) {
	entries := make([]domlog.Entry, 0, len(m.entries))
	for _, e := range m.entries {
		entries = append(entries, e)
	}
	return EntryList{Entries: entries, Total: len(entries)}, nil
}

func TestEventLogInterface(t *testing.T) {
	var el EventLog = &mockEventLog{entries: make(map[domlog.ID]domlog.Entry)}
	e := domlog.Entry{
		Action:    domlog.ActionStart,
		Session:   domlog.SessionWork,
		Payload:   map[string]any{"key": "value"},
		Timestamp: time.Now(),
	}
	id, err := el.AddEntry(context.Background(), e)
	if err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}
	list, err := el.ListEntries(context.Background(), ListParams{})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if list.Total != 1 {
		t.Errorf("Total = %d, want 1", list.Total)
	}
}

func TestErrNotFound(t *testing.T) {
	if ErrNotFound == nil {
		t.Error("ErrNotFound is nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/state/log/ -v`
Expected: FAIL — `undefined: EventLog`, `undefined: ListParams`, `undefined: EntryList`, `undefined: ErrNotFound`

- [ ] **Step 3: Write minimal implementation**

Create `internal/adapter/state/log/eventlog.go`:

```go
package log

import (
	"context"
	"errors"
	"time"

	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
)

type ListParams struct {
	Action        *domlog.Action
	Session       *domlog.Session
	From, To      *time.Time
	Limit, Offset int
}

type EntryList struct {
	Entries []domlog.Entry
	Total   int
}

type EventLog interface {
	AddEntry(ctx context.Context, entry domlog.Entry) (domlog.ID, error)
	ListEntries(ctx context.Context, params ListParams) (EntryList, error)
}

var ErrNotFound = errors.New("adapter: log entry not found")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/state/log/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(adapter): add log.EventLog port interface

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 4: Delete old `adapter/state/repository.go`

**Files:**
- Delete: `internal/adapter/state/repository.go`

- [ ] **Step 1: Delete the file**

Run: `rm internal/adapter/state/repository.go`

- [ ] **Step 2: Verify the build passes**

Run: `go build ./...`
Expected: Build fails — `port/state/sqlite/port.go` imports `adapter/state` which no longer exports `Repository`, `ListNotesParams`, `NoteList`.

This is expected: the SQLite port still references the old adapter package. The fix is in sub-plan 02-sqlite-ports. For now, the build failure confirms the old interface is gone and the new interfaces exist.

Run: `go build ./internal/domain/... ./internal/adapter/state/note/ ./internal/adapter/state/log/`
Expected: PASS — the new packages build independently.

- [ ] **Step 3: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
refactor(adapter): remove old Repository interface

SQLite port update deferred to 02-sqlite-ports sub-plan.

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`
