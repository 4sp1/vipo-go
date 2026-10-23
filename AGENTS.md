# AGENTS.md

## Project Overview

Vipo Server — a Go backend for a Pomodoro timer tracking application. Manages pomodoro sessions, notes, and timer event logs. API-first design with an OpenAPI spec driving endpoint definitions.

## Architecture

Hexagonal / Ports & Adapters pattern:

```
internal/
├── domain/state/          # Core domain models (zero external deps)
│   ├── pomodoro/state.go  # Session state enum
│   ├── note/note.go       # Note aggregate
│   └── log/entry.go       # Event log entries
├── adapter/state/
│   ├── note/store.go      # note.Store port interface + ErrNotFound
│   └── log/eventlog.go    # log.EventLog port interface + ErrNotFound
├── port/state/sqlite/     # SQLite implementation (shared *DB wrapper)
│   ├── db.go              # New(file) → *DB, NoteStore(), EventLog(), Migrate()
│   ├── note_store.go      # note.Store implementation
│   ├── event_log.go       # log.EventLog implementation
│   ├── converters.go      # *FromStorage enum converters
│   └── migrations.go      # goose v3 Provider runner
├── port/oapi/             # Generated OAPI types/server
│   ├── oapi_types.gen.go / oapi_server.gen.go
│   └── http/translate.go  # OAPI ↔ domain enum translation
├── svc/                   # Service layer (StrictServerInterface impl)
│   ├── service.go         # 6 handlers
│   └── errors.go          # Validation error types
├── app/                   # Application wiring
│   ├── apperr/            # Error code taxonomy + Wire()
│   ├── wire.go            # NewHandler → http.Handler
│   └── server.go          # NewServer → *Server
cmd/
└── vipo-server/main.go    # cobra CLI (serve + migrate)
migrations/
├── embed.go               # //go:embed *.sql
└── *.sql                  # goose-annotated migrations
```

**Dependency direction:** `app` → `svc` → `adapter` (interface) ← `port` (implementation), `domain` is imported by all.


### Key Design Decisions

- **Domain layer has zero imports** except stdlib — no external dependencies leak into domain models.
- `pomodoro.State` is an iota-based `int` enum (not a string), stored as integer in SQLite. Maps to `work=0, short_break=1, long_break=2`.
- `log.Entry` carries `ID`, is a plain struct with `Payload map[string]any` (free-form JSON). No sealed interface — new_note is excluded from log endpoints (handled by notes CRUD).
- `log.Action` and `log.Session` are separate types from `pomodoro.State` — they're distinct domain concepts even though they share similar values.
- SQLite uses `github.com/glebarez/go-sqlite` (pure-Go driver, no CGo required).
- Timestamps stored as `INTEGER` Unix seconds, parsed manually in the adapter layer (`time.Unix(...)` at read/write). Seconds precision (no nanoseconds). All times stored/queried in UTC.
- Empty slices return `[]note.State{}`/`[]log.Entry{}` not `nil` (see `ListNotes`/`ListEntries` in `port/state/sqlite/`).


## Commands

No Makefile or build scripts yet. Standard Go commands:

```bash
go build ./...           # Build all packages
go test ./...            # Run all tests
go vet ./...             # Vet
jj describe -m "msg"     # Commit (uses jj, not git directly)
jj new                   # Start new change after commit
```

## Database & Migrations

- **Migration tool:** Goose v3 — `migrations/` dir with `.sql` files using `-- +goose Up`/`-- +goose Down` annotations.
- **In-app runner:** `goose.NewProvider(goose.DialectSQLite3, db, fsys)` + `provider.Up(ctx)` runs on startup via `DB.Migrate(ctx)`. Migrations are embedded via `//go:embed` in `migrations/embed.go`.
- **Tables:** `notes (id INTEGER PK AUTOINCREMENT, note TEXT NOT NULL, pomodoro_state INTEGER NOT NULL, created_at INTEGER NOT NULL)` and `log_entries (id INTEGER PK AUTOINCREMENT, action INTEGER NOT NULL, session INTEGER NOT NULL, payload TEXT NOT NULL, timestamp INTEGER NOT NULL)`.
- Timestamps are Unix seconds stored as `INTEGER` in both tables.
- Database files are opened via `sql.Open("sqlite", file)` in `port/state/sqlite/db.go`; the `*DB` wrapper exposes `NoteStore()`, `EventLog()`, and `Migrate(ctx)`.

## API Design (openapi.json)

- OpenAPI 3.0.3 spec at project root — the canonical source of truth for HTTP endpoints.
- Currently only `notes` CRUD endpoints (`POST /notes`, `GET /notes`, `GET /notes/{id}`, `DELETE /notes/{id}`).
- In-progress: `POST /log/{action}` endpoints for timer events (plan in `docs/plans/`).
- `PomodoroState` enum in OpenAPI uses strings (`work`, `short_break`, `long_break`), but Go domain uses integers — translation happens at the HTTP/adapter boundary.

## Conventions

- **Commit style:** Ultra-compressed conventional commits via [caveman-commit](https://github.com/malikbenkirane/skilldrift) skill — subject ≤50 chars, body only when "why" isn't obvious. Styles: `feat(scope):`, `refactor(scope):`, `docs(plan):`, `chore(domain):`.
- **VCS:** Jujutsu (`jj`), not raw git. Use [core-commands](https://github.com/malikbenkirane/skilldrift) skill for shell commands, especially jj/git operations.
- **Package naming:** Packages are named after their domain concept (`note`, `pomodoro`, `log`), not after patterns (`models`, `entities`). Ports follow the same rule: they're named after the storage/aggregation concept they implement (`note.Store`, `log.EventLog` in the planned split structure).
- **Type naming:** Domain types use `State` for aggregate roots (e.g., `note.State`), enums use the type name directly (e.g., `pomodoro.State`, `log.Action`, `log.Session`). Enums stored as raw ints get explicit `*FromStorage` converters in `port/state/sqlite` (e.g. `PomodoroStateFromStorage`) instead of scanning DB ints directly into domain enums.
- **Error wrapping:** All errors use `fmt.Errorf("verb: %w", err)` pattern — no sentinel errors in domain; adapter-level typed errors (e.g. `note.ErrNotFound`) are declared in the adapter packages.
- **Constructor naming:** The SQLite adapter uses `func main(file string)` as constructor, not `New*` — a known wart, candidate for a boring `New*` rename.

## Gotchas

- `cmd/vipo-server/main.go` uses cobra with `serve` and `migrate` subcommands. Config via viper (flag > env `VIPD_*` > defaults).
- The svc layer implements `oapi.StrictServerInterface` and returns typed errors (not HTTP response objects) for failure cases. The app layer maps these to HTTP responses via `apperr.Wire()`.
- 500 errors are plain text (no JSON body), matching the spec which defines no 500 response. 400/404 errors use the `Error{code, message}` schema with enumerated `code` values.
- The `pomodoro_state` column stores Go's iota integer values (0, 1, 2), not the string names from OpenAPI. Ensure the translation layer handles this correctly when building HTTP handlers.
`log.Action` includes `ActionUnknown` (iota 0) as a zero-value sentinel; unknown/removed values are rejected at the storage boundary via converters.
- Out-of-range integers from SQLite must not flow silently into `pomodoro.State` — use validating `*FromStorage` converters at the storage boundary (see the discussion doc).
- Plans in `docs/plans/` follow a specific task format with checkboxes — use the `complete-plan-execution` skill when implementing them.

## OAPI Code Generation

- **Generator:** oapi-codegen v2 (managed via `go tool`)
- **Config files:** `internal/port/oapi/cfg.yaml` (types), `internal/port/oapi/strict.cfg.yaml` (strict server + std-http handler)
- **Generation command:** `go generate ./internal/port/oapi/`
- **Generated files:** `oapi_types.gen.go`, `oapi_server.gen.go` — do NOT edit these; regenerate instead
- **StrictServerInterface:** Router-agnostic Go interface. Service layer implements this to provide typed HTTP handlers.
- **Translation layer:** `internal/port/oapi/http/translate.go` converts between OAPI string enums and domain int enums.
- **Important:** Domain types use int iota (`pomodoro.State`, `log.Action`, `log.Session`). OAPI types use string enums (`PomodoroState`, `LogAction`). The translation layer bridges this gap.
