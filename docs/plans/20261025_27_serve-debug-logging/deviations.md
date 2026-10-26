# Deviations — 20261025_27_serve-debug-logging

## Sub-plan 01 (config and lifecycle)

- **One commit per sub-plan instead of per task.** Tasks 1 and 2 both touch
  `cmd/vipo-server/main.go`; the working-copy model of jj and the prohibition
  on `jj split` made atomic per-task commits impractical, so tasks 1–3 were
  committed as a single change. Verification steps ran unchanged per task.
- **Pre-existing `cmd/vipo-server/main_test.go` was not created fresh**;
  `TestResolveLogLevel` was appended to the existing file instead (plan assumed
  a create).
- **`migrate` flag block placement**: inserted after the `migrate` command
  definition rather than immediately next to `serve`'s (the plan's snippet
  would reference `migrate` before its declaration).

## Sub-plan 02 (http middleware)

- **Body-capture assertions**: the plan's note that raw `strings.Contains` on
  the payload works against `slog.TextHandler` output was wrong — TextHandler
  quotes/escapes the body value (`\"`), so tests assert against
  `strconv.Quote(payload)` (and the escaped error-body form). Semantics
  unchanged.
- **`TestMiddlewareBodiesAtDebugOnly/debug`**: the test handler now reads the
  request body (`io.ReadAll(r.Body)`) before responding. The tee pattern only
  fills the buffer when the handler reads it; the plan's test expected a
  captured body without reading.
- Implementation comments from the plan sketch (comment block on
  `accessRecorder`) omitted; behavior and API identical.
- One e2e test (`TestDebugModeLogsBodiesE2E`) exposes per-request WARN access
  lines from other e2e tests in the package's default logger output; harmless
  warnings on stderr during tests, matches plan's expected note.

## Sub-plan 03 (svc + storage debug)

- **`TestSvcEmitsDebugDiagnostics` setup**: uses a shared `svcWithTestDB`
  helper (`db.Close` via `t.Cleanup`) instead of the plan's inline
  create/migrate/defer; construction path identical to the plan
  (`sqlite.New(":memory:")` + `Migrate`).
- No storage warn-silence test was added (plan only specified the debug
  emission test for storage); warn-level silence is covered for svc and the
  HTTP middleware.

## Process

- One `heredoc`-append attempt on `internal/app/e2e_test.go` was made and
  immediately corrected to the Write-tool flow (content identical).
