# 03 — Migrations: Goose v3 Provider

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Atlas with goose v3 as the migration tool. Convert existing SQL files to goose-annotated format, embed them via `embed.FS`, and run migrations in-app on startup via the goose v3 Provider API.

**Architecture:** Migrations live in `migrations/` as goose-annotated SQL files, embedded into the binary via `go:embed`. The `*DB` wrapper from sub-plan 02 gains a `Migrate(ctx)` method that creates a `goose.NewProvider(goose.DialectSQLite3, db, fsys)` and calls `provider.Up(ctx)`. The goose runner shares the same `*sql.DB` as the ports. [S13], [S14], [S21].

**Tech Stack:** `github.com/pressly/goose/v3`, Go stdlib (`embed`, `io/fs`).

**Depends on:** Sub-plan 02 (`*DB` wrapper with `*sql.DB` field). Independent of sub-plan 01 (migration files don't reference domain types).

**Concurrent with:** Sub-plan 01 (different files, no conflicts).

---

## Files

- Modify: `migrations/20260517082423.sql`
- Modify: `migrations/20260614075800.sql`
- Delete: `migrations/atlas.sum`
- Modify: `internal/port/state/sqlite/db.go` (add `Migrate` method)
- Create: `internal/port/state/sqlite/migrations.go`
- Modify: `go.mod` (add goose v3)
- Modify: `AGENTS.md`

---

### Task 1: Add goose v3 dependency and convert migration files

**Files:**
- Modify: `go.mod`
- Modify: `migrations/20260517082423.sql`
- Modify: `migrations/20260614075800.sql`
- Delete: `migrations/atlas.sum`

- [ ] **Step 1: Add goose v3 dependency**

Run: `go get github.com/pressly/goose/v3@latest`

This adds `github.com/pressly/goose/v3` to `go.mod` and updates `go.sum`.

- [ ] **Step 2: Convert notes migration to goose format**

Replace the entire contents of `migrations/20260517082423.sql` with:

```sql
-- +goose Up
CREATE TABLE notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    note TEXT NOT NULL,
    pomodoro_state INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE notes;
```

- [ ] **Step 3: Convert log_entries migration to goose format**

Replace the entire contents of `migrations/20260614075800.sql` with:

```sql
-- +goose Up
CREATE TABLE log_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action INTEGER NOT NULL,
    session INTEGER NOT NULL,
    payload TEXT NOT NULL,
    timestamp INTEGER NOT NULL
);

-- +goose Down
DROP TABLE log_entries;
```

- [ ] **Step 4: Drop atlas.sum**

Run: `rm migrations/atlas.sum`

- [ ] **Step 5: Verify go.mod and build**

Run: `go mod tidy && go build ./...`
Expected: Build may fail on `internal/port/state/sqlite/` if sub-plan 02 hasn't been executed yet (old `port.go` references removed `adapter/state` types). Verify at least:

Run: `go vet github.com/pressly/goose/v3`
Expected: PASS (dependency is valid)

- [ ] **Step 6: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
chore(migrations): switch from Atlas to goose v3

Add goose-annotated Up/Down to existing SQL files. Drop
atlas.sum. Add pressly/goose/v3 dependency. [S13, S14]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 2: Embedded migrations and goose Provider runner

**Files:**
- Create: `internal/port/state/sqlite/migrations.go`
- Modify: `internal/port/state/sqlite/db.go` (add `Migrate` method)
- Test: `internal/port/state/sqlite/migrations_test.go`

- [ ] **Step 1: Write the failing test**

```go
package sqlite

import (
	"context"
	"testing"
)

func TestMigrateCreatesTables(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify notes table exists by inserting a row
	ns := db.NoteStore()
	id, err := ns.AddNote(context.Background(), testNote("migration test"))
	if err != nil {
		t.Fatalf("AddNote after migrate: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}

	// Verify log_entries table exists
	el := db.EventLog()
	logID, err := el.AddEntry(context.Background(), testLogEntry())
	if err != nil {
		t.Fatalf("AddEntry after migrate: %v", err)
	}
	if logID != 1 {
		t.Errorf("log id = %d, want 1", logID)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate 1: %v", err)
	}
	// Running again should not error — goose tracks applied versions
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate 2: %v", err)
	}
}
```

Note: `testNote` and `testLogEntry` are test helpers. If sub-plan 02 tests already define them in the same package, reuse them. Otherwise, define them in this test file:

```go
func testNote(text string) (n domnote.State) {
	n.Note = text
	n.Pomodoro = pomodoro.Work
	n.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return
}

func testLogEntry() (e domlog.Entry) {
	e.Action = domlog.ActionStart
	e.Session = domlog.SessionWork
	e.Payload = map[string]any{"test": true}
	e.Timestamp = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return
}
```

Include the necessary imports at the top of the test file:

```go
import (
	"context"
	"testing"
	"time"

	domlog "github.com/4sp1/vipo-go/internal/domain/state/log"
	domnote "github.com/4sp1/vipo-go/internal/domain/state/note"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)
```

If the helper functions conflict with existing helpers in the package (from sub-plan 02 tests), rename them or move them to a shared `testhelpers_test.go` file. The important thing is that both `testNote` and `testLogEntry` are available.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/port/state/sqlite/ -run TestMigrate -v`
Expected: FAIL — `undefined: db.Migrate`

- [ ] **Step 3: Write minimal implementation**

Create `internal/port/state/sqlite/migrations.go`:

```go
package sqlite

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (d *DB) Migrate(ctx context.Context) error {
	subFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("sub fs: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, d.db, subFS)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
```

Wait — the `//go:embed migrations/*.sql` directive resolves relative to the Go source file's directory, which is `internal/port/state/sqlite/`. The migrations live at the project root `migrations/`. The embed path must be within the module, and `//go:embed` cannot use `../` to escape the package directory.

Solution: create a symlink or move the embed directive to a package at the project root. The simplest approach is to create an `internal/migrations/` package that embeds the files, but `//go:embed` still can't reach `../../migrations/`.

Better solution: copy or move the SQL files into `internal/port/state/sqlite/migrations/` and embed from there. But the decisions doc says "Existing `.sql` files gain `-- +goose Up`/`-- +goose Down` annotations" — implying they stay in `migrations/`.

The cleanest approach: keep the SQL files in `migrations/` at the project root, but create a thin embed package that can reach them. In Go 1.26, `//go:embed` still cannot traverse upward with `../`. The standard pattern is to place the `//go:embed` directive in a file at or above the embedded directory.

Create `internal/port/state/sqlite/migrations.go` at the package directory, and instead of embedding from there, create a dedicated embed package at the project root:

Create `migrations/embed.go`:

```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

Then `migrations.go` in the sqlite package imports it:

```go
package sqlite

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/4sp1/vipo-go/migrations"
	"github.com/pressly/goose/v3"
)

func (d *DB) Migrate(ctx context.Context) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, d.db, migrations.FS)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
```

The `migrations` package at the project root embeds its own `*.sql` files via `//go:embed *.sql` — the directive resolves relative to `embed.go` in the same directory.

**Revised implementation:**

Create `migrations/embed.go`:

```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

Create `internal/port/state/sqlite/migrations.go`:

```go
package sqlite

import (
	"context"
	"fmt"

	"github.com/4sp1/vipo-go/migrations"
	"github.com/pressly/goose/v3"
)

func (d *DB) Migrate(ctx context.Context) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, d.db, migrations.FS)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
```

Note: `goose.NewProvider` accepts an `fs.FS` — `embed.FS` satisfies this interface directly, so no `fs.Sub` is needed because the embed directive matches `*.sql` in the `migrations/` directory root. [S13], [S14].

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/port/state/sqlite/ -run TestMigrate -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(port): add goose v3 Provider migration runner

migrations/embed.go embeds SQL files via //go:embed.
DB.Migrate creates goose.NewProvider with shared *sql.DB
and runs provider.Up. Idempotent via goose version
table. [S13, S14, S21]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 3: Update AGENTS.md

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update the Database & Migrations section**

In `AGENTS.md`, find the `## Database & Migrations` section and replace it with:

```markdown
## Database & Migrations

- **Migration tool:** Goose v3 — `migrations/` dir with `.sql` files using `-- +goose Up`/`-- +goose Down` annotations.
- **In-app runner:** `goose.NewProvider(goose.DialectSQLite3, db, fsys)` + `provider.Up(ctx)` runs on startup via `DB.Migrate(ctx)`. Migrations are embedded via `//go:embed` in `migrations/embed.go`.
- **Tables:** `notes (id INTEGER PK AUTOINCREMENT, note TEXT NOT NULL, pomodoro_state INTEGER NOT NULL, created_at INTEGER NOT NULL)` and `log_entries (id INTEGER PK AUTOINCREMENT, action INTEGER NOT NULL, session INTEGER NOT NULL, payload TEXT NOT NULL, timestamp INTEGER NOT NULL)`.
- Timestamps are Unix seconds stored as `INTEGER` in both tables.
- Database files are opened via `sql.Open("sqlite", file)` in `port/state/sqlite/db.go`; the `*DB` wrapper exposes `NoteStore()`, `EventLog()`, and `Migrate(ctx)`.
```

- [ ] **Step 2: Update the Gotchas section**

In `AGENTS.md`, find the Gotchas bullet about migrations:

```
- Database files are opened via `sql.Open("sqlite", file)` — no migration runner in-app; migrations are managed externally.
```

Replace with:

```
- Database files are opened via `sql.Open("sqlite", file)` in `port/state/sqlite/db.go`. Migrations run in-app via `DB.Migrate(ctx)` using goose v3 Provider — no external migration tool needed.
```

- [ ] **Step 3: Verify build and vet**

Run: `go vet ./migrations/... && go build ./migrations/...`
Expected: PASS

- [ ] **Step 4: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
docs(agents): update migration docs for goose v3

Replace Atlas references with goose v3 Provider. Document
in-app runner, embed pattern, and *DB.Migrate. [S14]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`
