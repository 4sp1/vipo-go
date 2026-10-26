# 01 — Config and lifecycle (issue 27) Sub-plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `complete-plan-execution` to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** `serve` accepts `--debug` and `--log-level` (combo hard-fails), sets the global slog default (JSON → stderr, warn default, CLI-flags-only override), and the lifecycle lines move to `fmt.Println`.

**Architecture:** Plain cobra flags (not viper-bound: the runtime level can only be changed via command-line flags, so no `VIPD_LOG_LEVEL`/`VIPD_DEBUG` env override). Level resolution is a pure, testable function `resolveLogLevel` in `package main`. `slog.SetDefault` is called at the top of `serve`'s and `migrate`'s `RunE` before any server is created.

**Tech Stack:** Go stdlib (`log/slog`, `os`), cobra.

---

### Task 1: Level resolution function `resolveLogLevel`

**Files:**
- Modify: `cmd/vipo-server/main.go`
- Test: `cmd/vipo-server/main_test.go` (create)

- [x] **Step 1: Write the failing test**

Create `cmd/vipo-server/main_test.go`:

```go
package main

import (
	"log/slog"
	"testing"
)

func TestResolveLogLevel(t *testing.T) {
	cases := []struct {
		name          string
		debugChanged  bool
		debug         bool
		levelChanged  bool
		lvl           string
		want          slog.Level
		wantErr       bool
	}{
		{"defaults", false, false, false, "", slog.LevelWarn, false},
		{"debug flag only", true, true, false, "", slog.LevelDebug, false},
		{"debug flag false", true, false, false, "", slog.LevelWarn, false},
		{"level debug", false, false, true, "debug", slog.LevelDebug, false},
		{"level info", false, false, true, "info", slog.LevelInfo, false},
		{"level warn", false, false, true, "warn", slog.LevelWarn, false},
		{"level error", false, false, true, "error", slog.LevelError, false},
		{"combo fails", true, true, true, "warn", 0, true},
		{"invalid value", false, false, true, "verbose", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveLogLevel(tc.debugChanged, tc.debug, tc.levelChanged, tc.lvl)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveLogLevel() err = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveLogLevel() err = %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveLogLevel() = %v, want %v", got, tc.want)
			}
		})
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run TestResolveLogLevel -v`
Expected: FAIL — `resolveLogLevel` (or `main_test.go`) does not exist / undefined.

- [x] **Step 3: Write minimal implementation**

In `cmd/vipo-server/main.go`, add below `newRootCmd`:

```go
func resolveLogLevel(debugChanged, debug bool, levelChanged bool, lvl string) (slog.Level, error) {
	if debugChanged && levelChanged {
		return 0, errors.New("--debug and --log-level are mutually exclusive; use only one")
	}
	if debugChanged {
		if debug {
			return slog.LevelDebug, nil
		}
		return slog.LevelWarn, nil
	}
	value := "warn"
	if levelChanged {
		value = lvl
	}
	switch value {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid --log-level %q: must be one of debug|info|warn|error", value)
	}
}
```

Extend the imports in `cmd/vipo-server/main.go` to include `"errors"`, `"log/slog"` (keep the existing ones).

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run TestResolveLogLevel -v`
Expected: PASS (all subtests).

- [x] **Step 5: Commit with caveman-commit**

Use `caveman-commit` for the message content, then the commit sequence from `core-commands`:

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file
4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

Message sketch: `feat(serve): add log-level resolution` (subject ≤50 chars).

---

### Task 2: Serve and migrate plumbing (flags + SetDefault + lifecycle lines)

**Files:**
- Modify: `cmd/vipo-server/main.go`
- Test: existing `go test ./cmd/...` still passes; behavioral verification via flag wiring below.

- [x] **Step 1: Define the flags on `serve` (and `migrate`) — no viper binding**

In `cmd/vipo-server/main.go`, right after the existing `serve.Flags().String("addr", ...)` line, add:

```go
	serve.Flags().Bool("debug", false, "Enable debug logging (sugar for --log-level=debug)")
	serve.Flags().String("log-level", "warn", "Log level (debug|info|warn|error)")

	migrate.Flags().Bool("debug", false, "Enable debug logging (sugar for --log-level=debug)")
	migrate.Flags().String("log-level", "warn", "Log level (debug|info|warn|error)")
```

Do NOT viper-bind these flags — `VIPD_LOG_LEVEL` and `VIPD_DEBUG` env vars must not set the level (decision from the grill session). The other flags (`db`, `addr`, `config`) keep their viper bindings.

- [x] **Step 2: Wire the level resolution + SetDefault into both RunE bodies, and swap lifecycle lines**

At the top of `serve`'s `RunE` (before `app.NewServer`), add:

```go
			debugVal, dbErr := cmd.Flags().GetBool("debug")
			if dbErr != nil {
				return fmt.Errorf("read --debug flag: %w", dbErr)
			}
			levelVal, lvErr := cmd.Flags().GetString("log-level")
			if lvErr != nil {
				return fmt.Errorf("read --log-level flag: %w", lvErr)
			}
			level, err := resolveLogLevel(
				cmd.Flags().Changed("debug"), debugVal,
				cmd.Flags().Changed("log-level"), levelVal,
			)
			if err != nil {
				return err
			}
			slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
			// ... existing body unchanged below
```

Do the same at the top of `migrate`'s `RunE` (same 12 lines, reading `cmd` of that subcommand).

Replace the lifecycle lines:

- In `serve` (currently `log.Printf("listening on %s", addr)`):

```go
			fmt.Printf("listening on %s\n", addr)
```

- In `migrate` (currently `log.Println("migrations complete")`):

```go
			fmt.Println("migrations complete")
```

Remove the `"log"` import from `cmd/vipo-server/main.go` (cobra prints returned RunE errors; returning the combo error makes cobra print `Error: <msg>` and `os.Exit(1)` follows in `main`, so the message reaches the user).

Note: `c.Flags().Changed("log-level")` reflects the actual CLI (`--log-level` given or not); the flag's own default (`"warn"`) supplies the value when unchanged. Do NOT viper-bind these flags — `VIPD_LOG_LEVEL` and `VIPD_DEBUG` env vars must not set the level (decision from the grill session). The existing flags (`db`, `addr`, `config`) keep their viper bindings unchanged.

- [x] **Step 3: Run tests / build**

Run: `go build ./... && go test ./... -count=1`
Expected: PASS everywhere.

Manual smoke (optional, for the reviewer):

```bash
go run ./cmd/vipo-server serve --debug &
# expect a JSON warning-free startup; lifecycle line `listening on ...` printed
kill %1
go run ./cmd/vipo-server serve --debug --log-level=error
# expect Error: --debug and --log-level are mutually exclusive; use only one
```

- [x] **Step 4: Commit with caveman-commit**

Use `caveman-commit` then the `core-commands` commit sequence: mktemp → Read temp file → Write message → `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`.

Message sketch: `feat(serve): wire log level flags, lifecycle lines` (subject ≤50 chars).

---

### Task 3: Doc fixes — real file names for storage call sites

**Files:**
- Modify: `docs/issues/27.md`
- Modify: `docs/drafts/22.md`

The issue says storage call sites live in `internal/port/state/sqlite/port.go` — that file does not exist; storage call sites live in `note_store.go` and `event_log.go`.

- [x] **Step 1: Fix `docs/issues/27.md` line 12**

Replace:

```
- **Layers logged:** HTTP (one middleware on `srv.Handler` in `internal/app/wire.go`), service (call sites in `internal/svc`), storage (call sites in `internal/port/state/sqlite/port.go`).
```

with:

```
- **Layers logged:** HTTP (one middleware on `srv.Handler` in `internal/app/wire.go`), service (call sites in `internal/svc`), storage (call sites in `internal/port/state/sqlite/note_store.go` and `event_log.go`).
```

- [x] **Step 2: Fix `docs/drafts/22.md` lines 48-49**

Replace:

```
- **Storage**: `slog.Debug` call sites in `internal/port/state/sqlite/port.go`
  for now. A generic `database/sql` driver-level logging wrapper is **out of
```

with:

```
- **Storage**: `slog.Debug` call sites in `internal/port/state/sqlite/note_store.go`
  and `event_log.go` for now. A generic `database/sql` driver-level logging
  wrapper is **out of
```

(keep the rest of the paragraph intact).

- [x] **Step 3: Run the full suite**

Run: `go test ./... -count=1`
Expected: PASS (docs-only change; sanity check nothing else broken).

- [x] **Step 4: Commit with caveman-commit**

Use `caveman-commit` then the `core-commands` commit sequence (mktemp → Read → Write → `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`).

Message sketch: `docs(issue): use real storage file names` (subject ≤50 chars).
