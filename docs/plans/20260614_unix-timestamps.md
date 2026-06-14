# Unix Timestamp Storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use complete-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change timestamp storage from RFC3339 strings to Unix integers for better storage efficiency and query performance.

**Architecture:** Update SQLite schema and adapter layer. Domain layer remains unchanged (uses `time.Time`). Adapter becomes responsible for Unix↔Time conversion at database boundary.

**Tech Stack:** Go, SQLite, Atlas migrations

---

## Task 1: Update Migration and Rehash

**Files:**
- Modify: `migrations/20260517082423.sql`
- Modify: `migrations/atlas.sum`

- [ ] **Step 1: Overwrite migration file**

Change `created_at` column type from `TEXT` to `INTEGER`:

```sql
CREATE TABLE notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note TEXT NOT NULL,
    pomodoro_state INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);
```

- [ ] **Step 2: Regenerate atlas.sum**

Run:
```bash
atlas migrate hash
```

Expected: `atlas.sum` updated with new checksum for `20260517082423.sql`

- [ ] **Step 3: Commit with caveman-commit**

```
refactor(migrations): store timestamps as unix integers

RFC3339 strings ~25 bytes vs int64 8 bytes.

Integer comparisons faster for ORDER BY and time-range queries.

Refs #6
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

## Task 2: Update AddNote Method

**Files:**
- Modify: `internal/port/state/sqlite/port.go:29-38`

- [ ] **Step 1: Change timestamp insertion from RFC3339 to Unix**

Replace the `AddNote` method to use `n.CreatedAt.Unix()` instead of `n.CreatedAt.Format(time.RFC3339)`:

```go
func (p *port) AddNote(ctx context.Context, n note.State) error {
	_, err := p.db.ExecContext(ctx,
		"INSERT INTO notes (note, pomodoro_state, created_at) VALUES (?, ?, ?)",
		n.Note, n.Pomodoro, n.CreatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert note: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Commit with caveman-commit**

```
refactor(adapter): write timestamps as unix ints in AddNote
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

## Task 3: Update ListNotes Method

**Files:**
- Modify: `internal/port/state/sqlite/port.go:40-91`

- [ ] **Step 1: Change timestamp parsing from RFC3339 to Unix**

Replace the scan and parse logic in `ListNotes`. Change the temporary variable from `string` to `int64` and use `time.Unix()` for conversion:

```go
func (p *port) ListNotes(ctx context.Context, params state.ListNotesParams) (state.NoteList, error) {
	baseQuery := "FROM notes"
	whereClause := ""
	args := []any{}

	if params.PomodoroState != nil {
		whereClause = " WHERE pomodoro_state = ?"
		args = append(args, *params.PomodoroState)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery + whereClause
	err := p.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return state.NoteList{}, fmt.Errorf("count notes: %w", err)
	}

	// Get paginated results
	query := "SELECT id, note, pomodoro_state, created_at " + baseQuery + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return state.NoteList{}, fmt.Errorf("query notes: %w", err)
	}
	defer rows.Close()

	var notes []note.State
	for rows.Next() {
		var n note.State
		var createdAtUnix int64
		err := rows.Scan(&n.ID, &n.Note, &n.Pomodoro, &createdAtUnix)
		if err != nil {
			return state.NoteList{}, fmt.Errorf("scan note: %w", err)
		}
		n.CreatedAt = time.Unix(createdAtUnix, 0)
		notes = append(notes, n)
	}

	if notes == nil {
		notes = []note.State{}
	}

	return state.NoteList{
		Notes: notes,
		Total: total,
	}, rows.Err()
}
```

- [ ] **Step 2: Commit with caveman-commit**

```
refactor(adapter): read timestamps as unix ints in ListNotes
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

## Task 4: Update GetNote Method

**Files:**
- Modify: `internal/port/state/sqlite/port.go:93-115`

- [ ] **Step 1: Change timestamp parsing from RFC3339 to Unix**

Replace the scan and parse logic in `GetNote`. Change the temporary variable from `string` to `int64` and use `time.Unix()` for conversion:

```go
func (p *port) GetNote(ctx context.Context, id note.ID) (note.State, error) {
	var n note.State
	var createdAtUnix int64

	err := p.db.QueryRowContext(ctx,
		"SELECT id, note, pomodoro_state, created_at FROM notes WHERE id = ?",
		id,
	).Scan(&n.ID, &n.Note, &n.Pomodoro, &createdAtUnix)

	if err != nil {
		if err == sql.ErrNoRows {
			return note.State{}, fmt.Errorf("note not found: %d", id)
		}
		return note.State{}, fmt.Errorf("get note: %w", err)
	}

	n.CreatedAt = time.Unix(createdAtUnix, 0)

	return n, nil
}
```

- [ ] **Step 2: Commit with caveman-commit**

```
refactor(adapter): read timestamps as unix ints in GetNote
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

## Task 5: Update AGENTS.md Documentation

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update timestamp documentation**

Find the line that says:
```
- Timestamps stored as `TEXT` in RFC3339 format, parsed manually in the adapter layer.
```

Replace with:
```
- Timestamps stored as `INTEGER` in Unix seconds, parsed manually in the adapter layer. Seconds precision (no nanoseconds). All times stored/queried in UTC.
```

- [ ] **Step 2: Commit with caveman-commit**

```
docs(agents): update timestamp storage convention
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

## Task 6: Verify Implementation

**Files:**
- None (verification only)

- [ ] **Step 1: Build all packages**

Run:
```bash
go build ./...
```

Expected: Success (no errors)

- [ ] **Step 2: Run tests (if any)**

Run:
```bash
go test ./...
```

Expected: Either PASS or "no test files" (no failures)

---

## Success Criteria

- [ ] All timestamp columns stored as `INTEGER` (Unix seconds)
- [ ] Adapter layer converts between Go's `time.Time` and integer at database boundary
- [ ] All times stored/queried in UTC (time.Unix() returns UTC by default)
- [ ] Existing Atlas migration overwritten, rehashed
- [ ] `AGENTS.md` updated to reflect new timestamp storage convention
- [ ] Code compiles without errors

## Notes

- The `log_entries` table already uses `INTEGER` for `timestamp` (see `migrations/20260614075800.sql:6`) — only `notes.created_at` needed migration.
- Unix timestamp storage uses `int64` for database scan, then `time.Unix(..., 0)` for seconds precision (nanoseconds set to 0).
- No tests exist in this codebase yet — `go test ./...` will report "no test files" which is acceptable.
- Migration strategy: overwrite existing migration file since no production data exists.
