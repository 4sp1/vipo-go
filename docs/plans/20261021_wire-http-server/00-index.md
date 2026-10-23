# Wire the HTTP Server — Index

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the HTTP server so the OpenAPI contract (`openapi.json`) can be exercised end-to-end.

**Architecture:** Hexagonal / Ports & Adapters. Split adapter into `note.Store` + `log.EventLog`; SQLite port shares one `*DB` connection; svc layer implements `StrictServerInterface`; app layer wires everything with cobra/viper. Goose v3 Provider runs SQL migrations on startup.

**Tech Stack:** Go 1.26, oapi-codegen v2 (strict server), glebarez/go-sqlite, goose v3 (SQL migrations), spf13/cobra + viper.

**Decisions doc:** `docs/drafts/20261021_wire_http_server_decisions.md` — all settled decisions with tag legend `[S1]`–`[S24]`.

**Pre-execution:** File `docs/issues/23.md` — "Investigate payload semantics via e2e + update OAPI spec" (referenced by [S22]).

---

Sub-plans (execution order):

- [ ] 01-domain-adapter — Update `log.Entry` (ID + payload map, delete Message); split `adapter/state` into `note.Store` + `log.EventLog`; typed `ErrNotFound`; `ListParams`/`EntryList`
- [ ] 02-sqlite-ports — Shared `*DB` wrapper; `*FromStorage` converters; note SQLite port (returns ID); log SQLite port (AddEntry, ListEntries w/ from/to/action/session filters + COUNT)
- [ ] 03-migrations-goose — goose v3 Provider; convert 2 SQL files to goose annotations; drop `atlas.sum`; `provider.Up` on startup
- [ ] 04-svc-layer — `StrictServerInterface` impl (6 handlers); translation; limit/offset/pomodoro_state/action/session/from/to validation; error mapping
- [ ] 05-app-config-main — `apperr.Code` taxonomy + `Wire()`; app wiring; cobra (serve + migrate) + viper (flag/env/config.json); main.go; HTTP error handler override
- [ ] 06-e2e-tests — app-level e2e (`httptest` + in-memory SQLite + goose); svc-level unit tests; shared DB test helper

Concurrent-eligible batches: `[01, 03]`, `[02]`, `[04, 05]`, `[06]`

Why this order and split: 01 and 03 are independent (domain/adapter types vs migration tooling). 02 needs 01 (implements the adapter interfaces). 04 needs 01+02 (svc calls ports). 05 needs 04 (app wires the svc) but its cobra/viper part is independent of 04's svc logic. 06 needs 05 (e2e tests the full stack). Each batch is the safe unit for one concurrent session; same-file edits stay within a batch.

**Commit convention:** Every commit uses the `core-commands` temp-file pattern (`mktemp` → Read → Write → `jj describe --stdin < <path> && rm <path> && jj new`). Messages follow `caveman-commit` style (subject ≤50 chars, conventional commits, body only when "why" isn't obvious).
