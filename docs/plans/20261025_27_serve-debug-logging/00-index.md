# serve --debug / --log-level (issue 27) — Index

> **For agentic workers:** REQUIRED SUB-SKILL: Use `complete-plan-execution` to implement each sub-plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--debug` / `--log-level` flags to `vipo-server serve` and make the HTTP, service, and storage layers observable via stdlib `slog` (JSON on stderr, warn default, anti-leak split).

Spec: `docs/issues/27.md`, decision record: `docs/drafts/22.md`.

Complexity gate: fired (estimated >600 single-file lines); split approved during the grill session.

Sub-plans (execution order):

- [ ] 01-config-and-lifecycle.md — cmd flags, level resolution, lifecycle lines, doc fixes
- [ ] 02-http-middleware.md — access middleware with debug-only body capture in `internal/app`
- [ ] 03-svc-storage-debug.md — `slog.Debug` call sites in svc and sqlite stores

Concurrent-eligible batches: `[01-config-and-lifecycle, 02-http-middleware, 03-svc-storage-debug]` — all three are concurrent-eligible together: they touch disjoint files (`cmd/vipo-server/`, `internal/app/`, `internal/svc/` + `internal/port/state/sqlite/`), and each is independently verified by `go test ./...`.

Why this order and split: one sub-plan per layer matches the decision to slice tasks per layer; file sets are disjoint so a batch is the safe unit for one concurrent session. Sequential execution is equally valid.

At execution start, create `deviations.md` in this directory and record every deviation there as it happens; when execution ends, write `pr.md` proposing the PR description.
