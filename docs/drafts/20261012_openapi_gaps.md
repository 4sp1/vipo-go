# OpenAPI implementation gaps

Discovered during the audit in `docs/issues/17.md`. The two specs
(vipo-go and `4sp1/vipo-flutter` @ c1a6ac2) are identical, so every gap
below is an *implementation* gap, not a contract mismatch. The HTTP layer
does not exist yet; this file is the requirements upstream for that layer.

## Gap 1 — `from`/`to` sub-second precision

**Evidence.** `from`/`to` arrive as RFC3339 strings (`date-time`), converted
in `internal/port/oapi/http/translate.go:107` (`TimeFromRFC3339`). Storage
is Unix seconds (`migrations/*.sql`, `port/state/sqlite/port.go`). No policy
exists for inputs carrying sub-second components, and no test exercises
either direction of the conversion at a boundary.

**Policy (accepted).** Truncate to the second, inclusive:

- Parse with `time.RFC3339`; reject inputs without a timezone offset.
- `from` truncates down (`floor`) to the second: `2025-01-01T10:00:00.750Z`
  becomes `1735722000` (inclusive from that second).
- `to` rounds up (`ceil`) to the second minus one nanosecond and is applied
  inclusive at second precision, i.e. `to=...10:05:00.999Z` includes the
  whole second 10:05:00.
- All stored/compared values are UTC.

**Test case specs (table-driven, no implementation yet).**

| case | input | stored/compared value |
|---|---|---|
| exact second | `2025-01-01T10:00:00Z` | `1735722000` |
| sub-second from | `2025-01-01T10:00:00.750Z` | `1735722000` |
| sub-second to | `2025-01-01T10:00:00.999Z` | includes second `1735722000` |
| no timezone | `2025-01-01T10:00:00` | parse error |
| non-UTC offset | `2025-01-01T11:00+02:00` | `1735722000` (UTC-normalized) |
| decimal second | `2025-01-01T10:00:00.000Z` | `1735722000` |

## Gap 2 — `payload` round-trip fidelity

**Evidence.** Generated type is `*map[string]any` (`oapi_types.gen.go:66-67`)
stored as `payload TEXT`. A `json.Marshal`/`json.Unmarshal` round-trip is
semantically lossless but not byte-faithful: numbers surface as `float64`
(`1e5` → `100000`, `1.0` → `1`), key order changes, unicode escapes normalize.

**Decision (accepted).** Use `json.Marshal` on write, `json.Unmarshal` into
`map[string]any` on read. The contract is a free-form JSON object;
byte-identity is explicitly not promised.

**Alternative (noted, rejected).** `json.RawMessage`: validate once, store
raw bytes, return verbatim. Preserves bytes but reintroduces a contract the
spec does not make.

**Test case specs.** Round-trip at the storage boundary:

- empty object `{}`; `null` fields; string with unicode escapes and
  multibyte characters; nested object two levels deep; array of mixed
  scalars; integer larger than `float64` exact range is out of contract —
  assert semantic (JSON-equal, not string-equal) equality instead;
  `map[string]any` with numeric values survives as JSON-equal numbers.

## Gap 3 — `total` field

**Evidence.** `NoteList`/`LogEntryList` carry `total *int`
(`oapi_types.gen.go:107,123`). No repository query path produces it.

**Decision (accepted).** Keep the field. Each list query composes a
`COUNT(*)` from the same `WHERE` clause used for the page; one extra query
per list request. If previously unset (`omitempty`) today, the
implementation must always set it (count applies to the unpaginated filter).

**Test case spec.** Given 7 notes, `limit=3` returns `3` notes with
`total=7`; empty result returns `total=0`, not `nil`.

## Gap 4 — `Error.code` values

**Evidence.** `Error {message, code}` exists in both specs; `code` is a
bare string with no enumerated values. No codepath produces one.

**Decision.** `code` belongs to the HTTP layer (clean architecture: the
service/domain layer must not know HTTP error shapings). The wiring plan
will define the enumerated set (e.g. `note_not_found`,
`invalid_pomodoro_state`) and document them in the spec at that point.
Testing: table-driven handler tests asserting schema-valid `Error` bodies
per failure class (400, 404).

## Gap 5 — No server wired

**Evidence.** `internal/svc/`, `internal/app/`, `cmd/vipo-server/` are
empty; `StrictServerInterface` has no implementer (tracked in
`docs/issues/22.md`). The contract cannot be exercised end-to-end until
wired.

**Scope note.** Wiring is out of scope of this audit; nothing here needs
code before server-wiring. Transitional static: `translate.go` enum helpers
pass unit tests (`translate_test.go`) but timestamp and payload cases from
Gap 1/2 are unexercised.

## Additionally audited — no gap

- Spec diff: zero differences vs flutter @ c1a6ac2 (structure, schemas,
  filters, enums, metadata).
- `GET /logs` `session` filter typed as `PomodoroState` is intentional
  semantics matching the domain's `log.Session` type.
