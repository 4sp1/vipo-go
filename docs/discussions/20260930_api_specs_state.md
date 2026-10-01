# Discussion — API spec & layer separation state (2026-09-30)

Follow-up to `docs/issues/17.md`. This is an inspection report of the current
code against two rules:

1. **Domain types must never leak into ports.** Every port works on its own
   types and converts explicitly via `ToDomain` / `FromDomain` when needed.
2. **There must be an adapter interface for the HTTP server** (currently
   missing) so the OAPI port is a distinct adapter, per issue 17.

## 1. Current dependency graph

```
domain  (note, pomodoro, log)            ← stdlib only, clean
adapter/state/repository.go             ← imports domain (ok: port definition)
port/state/sqlite/port.go               ← imports adapter + domain
port/oapi/  (generated types + server)  ← imports nothing internal
port/oapi/http/translate.go             ← imports domain + port/oapi
svc, app                                ← empty
```

## 2. Layer separation findings

### 2.1 What is correct

- **Domain purity**: `internal/domain/state/*` imports only stdlib. Good.
- **Enum translation discipline**: `translate.go` is a textbook from/to pair —
  `PomodoroStateToDomain/FromDomain`, `LogActionToDomain/FromDomain`,
  `SessionToDomain/FromDomain`, plus `TimeTo/FromRFC3339`. The OAPI string
  enums and domain int enums are distinct types; nothing is silently cast.
- **`log.Session` kept separate from `pomodoro.State`**: `SessionToDomain`
  maps OAPI `PomodoroState` → `log.Session`, not `pomodoro.State`. Respects
  the "distinct domain concepts" rule.
- **SQLite port has no OAPI imports**: `port/state/sqlite/port.go` works
  purely on domain types (`note.State`, `pomodoro.State`) and adapter types
  (`state.ListNotesParams`, `state.NoteList`). Good.

### 2.2 Violations / grey areas

1. **Direct enum scan builds domain types implicitly** —
   `port/state/sqlite/port.go:72` scans `created_at` raw and maps ints into
   `n.Pomodoro` via `Scan(&n.Pomodoro)`. Storage-level ints flow into the
   domain enum with no conversion step; an out-of-range DB value (e.g. `3`)
   would silently produce an invalid `pomodoro.State`. Missing: a
   `PomodoroStateFromStorage`-style validation/conversion at the adapter
   boundary (the same rule the HTTP layer already follows — the SQLite layer
   should mirror it).
2. **Adapter interface leaks SQLite concerns upward**: `AddNote` returns only
   `error` with "note not found" reported by string content
   (`port/state/sqlite/port.go:101`,
   `port/state/sqlite/port.go:122`). The port can't distinguish not-found
   from a real failure without string matching. Either the port definition
   needs a typed sentinel/detection mechanism, or the adapter is dishonest
   about its contract. Since we don't want sentinel errors in domain, the
   decision must happen in `adapter/state` (declared failure taxonomy).
3. **`adapter/state` is a notes-only port** but our spec contract includes
   `/logs`. There is no `adapter/state` interface (or repo query types) for
   `log.Entry`, so no adapter for the log port exists yet.

## 3. Missing server adapter interface (the big gap)

Nothing implements the generated `StrictServerInterface`, so the request
flow the spec implies does not exist:

```
HTTP → oapi strict handlers → [MISSING: svc implementing StrictServerInterface]
     → [MISSING: adapter interface for note+log] → port/sqlite
```

Concretely missing:

- An **adapter interface for the whole server surface** covering both
  `notes` and `logs` (or two ports, one per aggregate, which fits the
  existing `adapter/state` naming). It should own:
  - `AddNote`, `ListNotes` (exists), `GetNote`, `DeleteNote` (exists)
  - `AddLogEntry`, `ListLogEntries` (missing — query params `action`,
    `session`, `from`/`to` date-time filters, `limit`, `offset`)
- A `log_entry` aggregate/domain record or a query type carrying
  `from`/`to` as `time.Time` (domain) before conversion, keeping
  `date-time` parsing strictly at the HTTP boundary.
- The `svc` layer implementing `StrictServerInterface` whose only job is
  calling the port + translating with the existing `http/translate.go`
  helpers — no other logic.
- `app` wiring: constructor for sqlite port, service, and the generated
  router.

### Decision needed

Two shapes are possible, we should pick explicitly before building:

- **A. One port, server-shaped** (`adapter/state` extended):
  `Server interface` with note+log methods, converted once in `svc`.
- **B. Two ports, aggregate-shaped**: `Repository` (notes, exists) and a
  new `EventLog` port each with their own from/to types.

Recommendation: **B**, matching current conventions (ports named after
domain concept, one port per aggregate), with `from/to` needing a
`LogParams` type with `time.Time` fields.

### Settled decisions (grill-me, 2026-09-30)

Conclusions recorded here are ground basis for future plan files.

**Structure**

- Two ports, one per aggregate, in **split adapter packages**:
  `adapter/state/note` and `adapter/state/log` (mirroring
  `domain/state/{note,log}`).
- Ports renamed in favor of consistency; both are named after the storage
  concept they implement:
  - **`note.Store`** (notes; renamed from `Repository`)
  - **`log.EventLog`** (log entries)
- Log port is **append-only**: only `AddEntry` and `ListEntries` — no
  `GetEntry` / `DeleteEntry` (the spec defines no such endpoints).

**Interfaces**

```go
package note

type ListParams struct {
    PomodoroState *pomodoro.State
    Limit, Offset int
}

type List struct {       // entity name appears only where unambiguous
    Notes []note.State
    Total int
}

type Store interface {
    AddNote(ctx context.Context, note note.State) error
    ListNotes(ctx context.Context, params ListParams) (List, error)
    GetNote(ctx context.Context, id note.ID) (note.State, error)
    DeleteNote(ctx context.Context, id note.ID) error
}

var ErrNotFound = errors.New("adapter: note not found")
```

```go
package log

type ListParams struct {
    Action         *log.Action
    Session        *log.Session
    From, To       *time.Time   // UTC only, seconds precision
    Limit, Offset  int
}

type EntryList struct {
    Entries []log.Entry
    Total   int
}

type EventLog interface {
    AddEntry(ctx context.Context, entry log.Entry) error
    ListEntries(ctx context.Context, params ListParams) (EntryList, error)
}

var ErrNotFound = errors.New("adapter: log entry not found")
```

**Error taxonomy**

- Typed errors **declared in the adapter packages**, kept out of domain:
  `note.ErrNotFound`, `log.ErrNotFound` (one per package).
- Ports match failures via `errors.Is`; the svc layer translates to
  HTTP 404.

**Conversion discipline (from/to everywhere)**

- Enums: HTTP layer already converts via `translate.go`; the **SQLite
  port must do the same at the storage boundary** — scan into `int64`
  and convert through validating storage-to-domain functions instead of
  scanning raw DB ints into `pomodoro.State` directly.
- Converter names (unexported, inside `port/state/sqlite`):
  `PomodoroStateFromStorage(int64) (pomodoro.State, error)`,
  `LogActionFromStorage(int64) (log.Action, error)`,
  `LogSessionFromStorage(int64) (log.Session, error)`.
  Unknown storage values become errors.
- Time: `date-time` strings parse at the HTTP boundary; **`time.Time`
  is the port boundary type, UTC only, seconds precision** at this
  stage. Unix-second storage conversion stays inside the sqlite port.

## 4. Cross-check with issue 17 spec comparison

- Spec (local and flutter @ c1a6ac2) match exactly; nothing to fix.
- Trusted type coverage in translate.go matches every OAPI schema we
  generate (verified against `oapi_types.gen.go` Action constants).
- `total` field in NoteList/LogEntryList: repository already returns it
  (`COUNT(*)` query in `port/state/sqlite/port.go:53`) — worth keeping.
