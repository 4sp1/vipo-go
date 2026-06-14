# AGENTS.md

## Project Overview

Vipo Server — a Go backend for a Pomodoro timer tracking application. Manages pomodoro sessions, notes, and timer event logs. API-first design with an OpenAPI spec driving endpoint definitions.

## Architecture

Hexagonal / Ports & Adapters pattern:

```
internal/
├── domain/state/          # Core domain models (zero external deps)
│   ├── pomodoro/state.go  # Session state enum (Work, ShortBreak, LongBreak)
│   ├── note/note.go       # Note aggregate (ID, Note text, Pomodoro state, timestamp)
│   └── log/entry.go       # Event log entries (Action, Session, Payload, Timestamp)
├── adapter/state/
│   └── repository.go      # Repository interface + query types (port definition)
├── port/state/sqlite/
│   └── port.go            # SQLite implementation of Repository
├── svc/                   # Service layer (currently empty)
└── app/                   # Application wiring (currently empty)
```

**Dependency direction:** `app` → `svc` → `adapter` (interface) ← `port` (implementation), `domain` is imported by all.

### Key Design Decisions

- **Domain layer has zero imports** except stdlib — no external dependencies leak into domain models.
- `pomodoro.State` is an iota-based `int` enum (not a string), stored as integer in SQLite. Maps to `work=0, short_break=1, long_break=2`.
- `log.Entry` uses a sealed interface pattern (`Message` with `isMessage()` unexported method) for extensible event payloads. Currently only `MessageNewNote` exists.
- `log.Action` and `log.Session` are separate types from `pomodoro.State` — they're distinct domain concepts even though they share similar values.
- SQLite uses `go-sqlite` (pure-Go driver from `modernc.org/sqlite`), no CGo required.
- Timestamps stored as `INTEGER` in Unix seconds, parsed manually in the adapter layer. Seconds precision (no nanoseconds). All times stored/queried in UTC.
- Empty slices return `[]note.State{}` not `nil` (see `ListNotes` in `port.go`).

## Commands

No Makefile or build scripts yet. Standard Go commands:

```bash
go build ./...           # Build all packages
go test ./...            # Run all tests
go vet ./...             # Vet
jj describe -m "msg"     # Commit (uses jj, not git directly)
jj new                   # Start new change after commit
```

No tests exist yet.

## Database & Migrations

- **Migration tool:** Atlas — `migrations/` dir with `.sql` files + `atlas.sum` checksum.
- **Single table so far:** `notes (id INTEGER PK AUTOINCREMENT, note TEXT NOT NULL, pomodoro_state INTEGER NOT NULL, created_at TEXT NOT NULL)`.
- The `log_events` table for `log.Entry` has not been created yet — only the domain model exists in code.
- Database files are opened via `sql.Open("sqlite", file)` — no migration runner in-app; migrations are managed externally.

## API Design (openapi.json)

- OpenAPI 3.0.3 spec at project root — the canonical source of truth for HTTP endpoints.
- Currently only `notes` CRUD endpoints (`POST /notes`, `GET /notes`, `GET /notes/{id}`, `DELETE /notes/{id}`).
- In-progress: `POST /log/{action}` endpoints for timer events (plan in `docs/plans/`).
- `PomodoroState` enum in OpenAPI uses strings (`work`, `short_break`, `long_break`), but Go domain uses integers — translation happens at the HTTP/adapter boundary.

## Conventions

- **Commit style:** Ultra-compressed conventional commits via [caveman-commit](https://github.com/malikbenkirane/skilldrift) skill — subject ≤50 chars, body only when "why" isn't obvious. Styles: `feat(scope):`, `refactor(scope):`, `docs(plan):`, `chore(domain):`.
- **VCS:** Jujutsu (`jj`), not raw git. Use [core-commands](https://github.com/malikbenkirane/skilldrift) skill for shell commands, especially jj/git operations.
- **Package naming:** Packages are named after their domain concept (`note`, `pomodoro`, `log`), not after patterns (`models`, `entities`).
- **Type naming:** Domain types use `State` for aggregate roots (e.g., `note.State`), enums use the type name directly (e.g., `pomodoro.State`, `log.Action`, `log.Session`).
- **Error wrapping:** All errors use `fmt.Errorf("verb: %w", err)` pattern — no sentinel errors, no custom error types yet.
- **Constructor naming:** The SQLite adapter uses `func main(file string)` as constructor, not `New*` — this is unconventional and may be a placeholder.

## Gotchas

- `cmd/vipo-server/` and `internal/svc/` and `internal/app/` directories exist but are **empty** — no main entrypoint or HTTP handler wiring yet. The project is domain-model-first, and the server/wiring layer hasn't been built.
- The `pomodoro_state` column stores Go's iota integer values (0, 1, 2), not the string names from OpenAPI. Ensure the translation layer handles this correctly when building HTTP handlers.
- `log.Action` includes `ActionUnknown` (iota 0) and `ActionNewNote` — these don't map to OpenAPI endpoints: `ActionUnknown` is a zero-value sentinel, and `ActionNewNote` is handled by the notes CRUD, not a log endpoint.
- Plans in `docs/plans/` follow a specific task format with checkboxes — use the `complete-plan-execution` skill when implementing them.