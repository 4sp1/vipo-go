# Wire the HTTP server — settled decisions (2026-10-21)

Ground basis for the implementation plan addressing `docs/issues/22.md`
("Wire the HTTP server"). Produced by a grill-me session against
`docs/issues/17.md`, `docs/drafts/20261012_openapi_gaps.md`, and
`docs/discussions/20260930_api_specs_state.md`. Every decision below is
settled; the plan files in `docs/plans/20261021_wire-http-server/` implement
them and must not silently re-open any branch.

Tag legend: `[Sn]` tags are referenced by the plan's `00-index.md` and by
sub-plan tasks for traceability.

## 1. Adapter architecture

**[S1] Two split adapter packages.** `adapter/state` splits into
`adapter/state/note` and `adapter/state/log`, mirroring
`domain/state/{note,log}`. Ports named after the aggregate they serve, per
the project's existing naming convention.

**[S2] Port names and surface.** `note.Store` (renamed from `Repository`)
and `log.EventLog` (new). The log port is append-only: `AddEntry` and
`ListEntries` only — no `GetEntry`/`DeleteEntry`, because the spec defines no
such endpoints.

**[S3] Typed not-found errors.** `note.ErrNotFound` and `log.ErrNotFound`
declared in their respective adapter packages, matched via `errors.Is`.
Domain stays sentinel-free; the svc translates not-found to HTTP 404.

**[S4] Storage-boundary enum converters.** Unexported
`PomodoroStateFromStorage`, `LogActionFromStorage`, `LogSessionFromStorage`
in `port/state/sqlite` scan raw `int64` from the DB and validate before
returning a domain enum. Unknown storage values become errors. No raw DB int
is ever scanned directly into a domain enum.

**[S9] Restructuring is in scope.** The adapter restructuring is part of the
wiring plan, not a separate prior plan, because the log port does not exist
yet and the wiring depends on it. Wiring against the current single
`Repository` shape is explicitly rejected — that shape is already agreed to be
abandoned.

### Interface shapes

```go
// adapter/state/note
package note

type ListParams struct {
    PomodoroState *pomodoro.State
    Limit, Offset int
}

type List struct {
    Notes []note.State
    Total int
}

type Store interface {
    AddNote(ctx context.Context, note note.State) (note.ID, error)
    ListNotes(ctx context.Context, params ListParams) (List, error)
    GetNote(ctx context.Context, id note.ID) (note.State, error)
    DeleteNote(ctx context.Context, id note.ID) error
}

var ErrNotFound = errors.New("adapter: note not found")
```

```go
// adapter/state/log
package log

type ListParams struct {
    Action        *log.Action
    Session       *log.Session
    From, To      *time.Time // UTC only, seconds precision
    Limit, Offset int
}

type EntryList struct {
    Entries []log.Entry
    Total   int
}

type EventLog interface {
    AddEntry(ctx context.Context, entry log.Entry) (log.ID, error)
    ListEntries(ctx context.Context, params ListParams) (EntryList, error)
}

var ErrNotFound = errors.New("adapter: log entry not found")
```

## 2. Domain model

**[S10] `log.Entry` gains an ID.** `type ID int64` in the `log` package;
`log.Entry` gets an `ID ID` field. The DB autoincrements `log_entries.id`;
`AddEntry` returns the assigned id. Mirrors `note.State` which already carries
`note.ID`.

**[S11] `Payload` becomes `map[string]any`; dead code deleted.** The
`Message` sealed interface and `MessageNewNote` struct are deleted — the only
variant (`new_note`) is excluded from log endpoints and handled by the notes
API. `log.Entry.Payload` is `map[string]any` (stdlib, domain stays pure) to
match the free-form contract directly. One type flows end-to-end; no
adapter-side record type or double translation.

**[S11 future work]** The real issue is the OAPI spec under-specifies
`payload`. Revisiting the spec (and the layer decoupling) is deferred until
payload semantics are understood via e2e testing. Tracked as a new issue
(`docs/issues/23.md`, to be filed) — see [S22].

**[S12] `AddNote` returns the id.** Signature becomes
`AddNote(ctx, note.State) (note.ID, error)`. The svc sets
`CreatedAt = time.Now().UTC()` before calling; the port returns the
autoincremented id. Matches the existing `AddNote` body which already reads
`n.CreatedAt.Unix()` (expects `CreatedAt` pre-set). Svc is time-authority,
port is id-authority.

**[S20] Keep `ActionUnknown`/`SessionUnknown` sentinels; remove
`ActionNewNote`.** The zero value of `log.Action` must be a clearly-invalid
marker, not a valid action like `ActionStart`. Keeping the `Unknown` sentinels
at iota 0 means a zero-value `log.Entry` fails validation immediately — a
safety guard against uninitialised values flowing through storage and the
wire. `ActionNewNote` (the last const) is removed as dead code; since it's
the last iota, no existing values 0–6 shift. `FromStorage` rejects `0`
(unknown) for both Action and Session, and rejects the removed `new_note`.

## 3. Storage & migrations

**[S5] Port boundary type is `time.Time` (UTC, seconds).** `date-time`
strings parse at the HTTP boundary only. `from`/`to` truncate-to-second
inclusive: `from` floors, `to` ceils to the whole second. All stored/compared
values are UTC. Unix-second `INTEGER` storage conversion stays inside the
sqlite port. (Gap 1 policy.)

**[S6] `payload` round-trip.** `json.Marshal` on write, `json.Unmarshal` into
`map[string]any` on read. Semantically lossless, not byte-faithful (numbers
surface as `float64`, key order changes). Byte-identity is explicitly not
promised. (Gap 2 decision.)

**[S7] `total` via `COUNT(*)`.** Each list query composes a `COUNT(*)` from
the same `WHERE` clause used for the page — one extra query per list request.
Always set (never `nil`/`omitempty`); count applies to the unpaginated
filter. (Gap 3 decision.)

**[S13/S14] In-app goose v3 Provider; goose-annotated SQL.** Migrations run
on startup via `goose.NewProvider(goose.DialectSQLite3, db, fsys)` +
`provider.Up(ctx)` (the post-2023 Provider API, not the globals). Existing
`.sql` files gain `-- +goose Up`/`-- +goose Down` annotations; `atlas.sum`
is dropped. AGENTS.md's "no in-app runner" convention is updated. SQL (not
Go) migrations, because the existing DDL is pure schema with no logic.

**[S14 future work]** The migration-tool change (Atlas → goose) and its
consequences are documented in the plan and in an AGENTS.md update.

**[S21] Shared `*DB` wrapper.** Single `New(file string) (*DB, error)` in
`port/state/sqlite` returns a `*DB` exposing `.NoteStore()` and `.EventLog()`
methods over one shared `*sql.DB`. Both ports and the goose runner hit the
same SQLite file; centralising connection lifecycle avoids three independent
connections to the same DB. Replaces the `func main(file string)` wart
flagged in AGENTS.md.

## 4. HTTP layer & validation

**[S8] `Error.code` is HTTP-layer.** The service/domain layer must not know
HTTP error shapings. The enumerated `code` set is defined in this wiring
plan. (Gap 4.)

**[S16] `apperr.Code` at the HTTP boundary.** The `Code int` enum (with
stringer) lives at the HTTP boundary, not in svc. Svc returns adapter errors
(`note.ErrNotFound`) and validation errors; the HTTP layer maps them to
`Code` → `Error{code, message}`. A separate `Wire()` switch produces
snake_case wire strings (stringer derives from Go identifiers and can't
emit `note_not_found`). `Code.Error()` delegates to `String()` for logging.

Enumerated codes (per failure class): `note_not_found`, `log_not_found`,
`invalid_request`, `invalid_pomodoro_state`, `invalid_log_action`,
`invalid_limit`, `invalid_offset`, `invalid_timestamp`.

**[S15/S24] `limit`/`offset` defaults and rejection.** `nil` `limit` → 50,
`nil` `offset` → 0 (spec defaults the Flutter client depends on). Explicitly
out-of-range values (`limit` < 1 or > 100; `offset` < 0) → 400 with
`Error{code: "invalid_limit"|"invalid_offset"}`. Defaults are applied in the
svc; rejection is the svc's job; the port receives validated `int`.

**[S23] 500s stay plain text.** The generated strict handler's default
`ResponseErrorHandlerFunc` emits `http.Error(w, err.Error(), 500)` — plain
text, no JSON body. This matches the spec, which defines no 500 response for
any endpoint. Gap 4 scoped `Error.code` to 400/404 only. Shaping 500s as
JSON is a separate spec-edit decision, not part of this plan.

## 5. App, tooling & plan structure

**[S17] cobra + viper; `serve` + `migrate` subcommands.** `cmd/vipo-server`
is a cobra root with two subcommands: `serve` (run the HTTP server) and
`migrate` (run migrations independently). Config precedence: flag > env >
config.json > defaults. viper handles the layered config in one dependency.

**[S18] Two test layers.** Svc-level unit tests (translation + validation
logic; fast, no DB) isolate logic failures. App-level e2e (`httptest` +
in-memory SQLite + goose `Provider.Up`) validates the contract end-to-end
and discovers payload semantics for the [S11] future-work issue. A shared
test helper sets up the temp DB.

**[S19] Six sub-plans.** The plan exceeds the 600-line / 15-task complexity
gate; output is a directory.

```
docs/plans/20261021_wire-http-server/
  00-index.md
  01-domain-adapter.md      — log.Entry (ID + payload map, delete Message); split adapter packages; note.Store + log.EventLog; typed ErrNotFound; ListParams/EntryList
  02-sqlite-ports.md        — shared *DB wrapper; storage converters; move note port; new log SQLite port (AddEntry, ListEntries w/ from/to/action/session filters + COUNT); constructor rename
  03-migrations-goose.md   — goose v3 Provider; convert 2 SQL files to goose annotations; drop atlas.sum; provider.Up on startup
  04-svc-layer.md           — StrictServerInterface impl (6 handlers); translation; limit/offset/pomodoro_state/action/session/from/to validation; error mapping
  05-app-config-main.md     — apperr.Code taxonomy + Wire(); app wiring; cobra (serve + migrate) + viper (flag/env/config.json); main.go
  06-e2e-tests.md           — app-level e2e (httptest + in-memory SQLite + goose); svc-level unit tests; shared DB test helper
```

Concurrent batches: `[01, 03]` → `[02]` → `[04, 05]` → `[06]`. 01 and 03 are
independent; 02 needs 01; 04 needs 01+02; 05 needs 04 (wiring) but its
cobra/viper part is independent of 04's svc logic; 06 needs 05. Each batch is
the safe unit for one concurrent session; same-file edits stay within a batch.

**[S22] Payload future-work issue.** A new `docs/issues/23.md` —
"Investigate payload semantics via e2e + update OAPI spec" — is filed as a
tracked issue and referenced by the plan's `00-index.md`. The deferral is
visible in the backlog, not buried in an executed plan. Rationale: the OAPI
`payload` field is under-specified; understanding what the Flutter client
actually sends requires e2e exercise, then the spec (and possibly the domain
layer decoupling) is revisited.

## References

- Issue: `docs/issues/22.md` — Wire the HTTP server.
- Upstream issue with conclusions: `docs/issues/17.md`.
- Gap document: `docs/drafts/20261012_openapi_gaps.md` (Gaps 1–5).
- Prior grill-me decisions: `docs/discussions/20260930_api_specs_state.md`.
- goose v3 Provider reference:
  https://pressly.github.io/goose/blog/2023/goose-provider/
