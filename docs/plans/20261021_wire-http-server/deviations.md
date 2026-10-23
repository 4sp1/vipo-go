# Execution Deviations — 20261021_wire-http-server

Deviations from the plan observed while executing sub-plans 01–06.

- **Sub-plan order:** 03's Task 1 (`go get goose`, SQL conversion, `atlas.sum` removal) was committed before 02 because `go mod tidy` was blocked by the stale `internal/port/state/sqlite/port.go`, which still imported the deleted `adapter/state` package. `go mod tidy` was completed only after 02 removed `port.go`.
- **02 Tasks 3–5 combined:** the note and log SQLite ports were committed in a single commit (`feat(port): implement note and log SQLite ports`), since `db.go` referencing both unimplemented types made partial package compilation impossible. `port.go` was deleted early (during 02 Task 1) to unblock `go mod tidy`.
- **Converters switch style:** `LogActionFromStorage`/`LogSessionFromStorage` switch on `log.Action(v)`/`log.Session(v)` instead of raw `int64` cases in the plan snippet (plan's snippet did not compile — mismatched typed constants against `int64` switch).
- **03 goose filename requirement:** migration files were renamed to include a description (`20260517082423_create_notes.sql`, `20260614075800_create_log_entries.sql`) — goose v3 Provider reported "no migrations found" for files lacking the `_description` suffix.
- **04 tasks combined:** all six `StrictServerInterface` handlers were implemented and committed in one commit rather than six, since the compile-time `var _ oapi.StrictServerInterface` assertion forces the full interface into each intermediate build.
- **04 svc stubs:** the plan's recording stubs ignored `ListParams`, making its own limit/filter tests unpassable; the stubs implement filtering, limit, and offset.
- **04 invalid-enum test:** plan's test expected a Go error for an invalid `pomodoro_state` while the implementation returns a 400 response object per [S8]; the test was fixed to assert the 400 response type (later superseded: see next).
- **05 error-return refactor:** per 05's own resolution, the svc was refactored (as its Task 3 requires) to return typed errors (`ErrInvalid*`, adapter `ErrNotFound`) instead of 400/404 response objects. The custom `listNotes400Response`/`listLogEntries400Response` visitor types were therefore never needed and their tests assert error types. Sub-plan 04 and 05 handler shapes conflict; 05's interpretation ([S16]) won.
- **05 `NewServer` signature:** dropped the unused `addr` parameter (plan passed it but never used it); the ErrorHandler interface/defaultErrorHandler indirection was also dropped as dead weight since `NewHandler` embeds its deserialize/serialize error handling.
- **06 commits combined:** e2e helper, e2e tests, and svc boundary tests were committed together (helper alone cannot compile as "used" without tests, and tasks 2–4 are one verification surface).
- **06 note:** 06's plan text still contains leftover `resp`(400-response-object) assumptions in some snippets; adjusted to typed-error assertions per the 05 refactor.
