# feat: serve --debug / --log-level

Fixes #27

## Summary

Make the server observable via stdlib `log/slog`: `serve` and `migrate` gain `--debug` and `--log-level` flags (mutually exclusive, CLI-flags-only — no viper/env override), setting the global slog default to JSON on stderr with `warn` as default. HTTP, service, and storage layers emit diagnostics, with an anti-leak split: only access metadata is visible at/above warn; request and response bodies (including error bodies) are debug-only.

## Changes

- `cmd/vipo-server`: pure `resolveLogLevel` (table-tested in `main_test.go`); `--debug`/`--log-level` flags on `serve` and `migrate` (not viper-bound); `slog.SetDefault(JSON→stderr)` at the top of both `RunE`s; lifecycle lines (`listening on ...`, `migrations complete`) moved to `fmt.Print*`; `log` import removed.
- `internal/app/logging.go`: `LoggingMiddleware` on `srv.Handler` logs access metadata (`method`/`path`/`status`/`duration_ms`) at WARN for every request including unmatched routes; captures request (dot-io tee) and response (recorder wrapper) bodies, logged at DEBUG only.
- `internal/svc` + `internal/port/state/sqlite`: one `slog.Debug` entry line per public handler/store method with identifying params (id, limit/offset, filters), before validation.
- Docs: storage call-site file names corrected in `docs/issues/27.md` (#27) and `docs/drafts/22.md` (#22). 
- Tests: middleware unit + warn-silence (anti-leak) tests, e2e debug-body test on the real handler, svc and sqlite emission tests + svc warn-silence test.
- Deviations from the plan are recorded in `docs/plans/20261025_27_serve-debug-logging/deviations.md`.

## Verification

`go build ./...`, `go test ./... -count=1`, `go vet ./...` all pass; manual smoke confirmed the lifecycle line appears with `--debug` and the combo `--debug --log-level=error` fails with the mutual-exclusion error.
