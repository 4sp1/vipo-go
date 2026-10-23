# Wire the HTTP server (notes + logs)

Closes #22. Refs #26 (follow-up: payload semantics).

## Summary

Wires the OpenAPI contract (`openapi.json`) end to end: domain models, SQLite adapter ports, goose-managed migrations, the service layer implementing `StrictServerInterface`, app-level wiring (error mapping, HTTP server), a cobra CLI (`serve` / `migrate`), and a full e2e test suite.

The PR is split into six reviewable layers (each landed as one or more atomic jj commits):

1. **Domain & adapter split** — `log.Entry` gains `ID` and a free-form `Payload map[string]any` (sealed `Message` interface deleted — `new_note` is excluded from log endpoints). The old single `adapter/state` repository is split into `note.Store` and `log.EventLog`, each with a typed `ErrNotFound`.
2. **SQLite ports** — shared `*DB` wrapper (`New(file)`, `NoteStore()`, `EventLog()`); storage-boundary converters (`PomodoroStateFromStorage`, `LogActionFromStorage`, `LogSessionFromStorage`) reject unknown/out-of-range ints so raw DB values never flow into domain enums. `AddNote`/`AddEntry` return autoincremented IDs; list endpoints filter + COUNT(*) totals; not-found maps to typed `ErrNotFound`.
3. **Migrations (Atlas → goose v3)** — `.sql` files converted to `-- +goose Up/Down` format with `_description` suffixes, `atlas.sum` removed, `migrations/embed.go` embeds the files, and `DB.Migrate(ctx)` runs `provider.Up` in-app (idempotent, version-tracked by goose).
4. **Service layer** — all six `StrictServerInterface` handlers (notes CRUD, log list/create). Svc is the time authority (`time.Now().UTC()`); ports are the ID authority. Enum translation via `port/oapi/http`. `limit` defaults 50 and is validated `[1,100]`, `offset >= 0`; `from` floors and `to` ceils to whole seconds. Per the settled decision [S16], svc returns **typed errors** (`ErrInvalid*`, adapter `ErrNotFound`) instead of HTTP response objects.
5. **App layer** — `apperr.Code` taxonomy + `Wire()` mapping svc errors to `{code, message, status}`. `NewHandler` produces plain-text 500s (spec defines no 500 body) and JSON `Error{code, message}` 400/404s. `NewServer(dbFile)` glues SQLite + svc + handler. `cmd/vipo-server` provides cobra `serve` (migrate + listen) and `migrate` subcommands, with viper layered config (flag > env `VIPD_*` > defaults).
6. **Tests** — app-level e2e via `httptest` + in-memory SQLite + goose: full HTTP contract for notes and logs (CRUD, filters, pagination, validation error codes, payload JSON round-trip). Svc-level boundary tests for from/to second-truncation and limit bounds. Unit tests for every new package.

## Testing

```
go test ./...   # all packages pass (unit + ports + svc + app e2e)
go vet ./...    # clean
go build ./...  # clean
```

E2e highlights asserted in `internal/app/e2e_test.go`:
- `POST /notes` → 201 with server-assigned ID/created_at; invalid `pomodoro_state` → 400 `code=invalid_pomodoro_state`.
- `GET /notes/{id}` / `DELETE /notes/{id}` → 200/204 and 404 with `code=note_not_found`.
- `GET /notes` and `GET /logs`: default limit 50, `limit`/`offset` validation codes (`invalid_limit`, `invalid_offset`), state/session filters, action+session combined filters, wide time range.
- `POST /logs` payload (nested maps, arrays, unicode) round-trips semantically through SQLite JSON storage; nil payload stays nil.
- `from`/`to` inclusive second-boundary behavior verified at the svc level.

## Deviations

Recorded in `deviations.md` (same directory). Highlights:
- Sub-plan order adjusted: 03's dependency conversion was committed before 02 because `go mod tidy` was blocked by the stale `port.go` (deleted early in 02).
- goose v3 requires a `_description` suffix in migration filenames — the two files were renamed accordingly.
- Sub-plan 04 and 05 conflict on error shaping (400 response objects vs typed errors); 05's interpretation of [S16] was implemented and 04's helpers/tests adjusted.
- Related tasks were combined into single commits where a partial package could not compile (e.g. both ports, all six handlers).
