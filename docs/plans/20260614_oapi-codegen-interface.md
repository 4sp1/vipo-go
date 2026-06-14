# Generate Server Interface from openapi.json with oapi-codegen

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate a typed, spec-driven `StrictServerInterface` from `openapi.json` using oapi-codegen, plus a translation layer between generated OAPI types and domain types.

**Architecture:** oapi-codegen reads `openapi.json` and produces Go types + a strict server interface under `internal/port/oapi/`. A hand-written translation layer in `internal/port/oapi/http/` converts between generated string-based enums (`PomodoroState`, `LogAction`) and domain int-based enums (`pomodoro.State`, `log.Action`), plus RFC3339 ↔ `time.Time` helpers. Two generation configs: one for models, one for the strict server interface.

**Tech Stack:** Go 1.26.2, oapi-codegen v2, gorilla/mux (required by strict-server generation; `StrictServerInterface` itself is router-agnostic), github.com/oapi-codegen/runtime

---

## File Structure

```
internal/port/oapi/
├── cfg.yaml                  # oapi-codegen config: types only
├── strict.cfg.yaml           # oapi-codegen config: strict server + gorilla handler
├── gen.go                    # go:generate directives (package oapi)
├── oapi_types.gen.go         # generated — models (Note, NoteList, CreateNoteRequest, PomodoroState, Error, LogAction, CreateLogEntryRequest, LogEntry, LogEntryList)
└── oapi_server.gen.go        # generated — StrictServerInterface, request/response objects, gorilla handler

internal/port/oapi/http/
├── translate.go              # PomodoroState ↔ pomodoro.State, LogAction ↔ log.Action, Session ↔ log.Session, RFC3339 ↔ time.Time
└── translate_test.go         # tests for all conversion functions
```

---

### Task 1: Add oapi-codegen Tool Dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add oapi-codegen as a Go tool dependency**

Run:
```bash
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

This adds `oapi-codegen` to the `tool` directive in `go.mod` (Go 1.24+ feature).

- [ ] **Step 2: Add oapi-codegen/runtime dependency**

Run:
```bash
go get github.com/oapi-codegen/runtime
```

The strict server generated code imports `github.com/oapi-codegen/runtime` for strict handler types.

- [ ] **Step 3: Add gorilla/mux dependency**

Run:
```bash
go get github.com/gorilla/mux
```

The strict server config uses `gorilla-server: true` — the generated handler code imports `github.com/gorilla/mux`. The `StrictServerInterface` itself is router-agnostic.

- [ ] **Step 4: Verify dependencies compile**

Run:
```bash
go build ./...
```

Expected: Success (no errors)

- [ ] **Step 5: Commit with caveman-commit**

```
chore(deps): add oapi-codegen, runtime, gorilla/mux
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

### Task 2: Create oapi-codegen Config Files and Generate Directive

**Files:**
- Create: `internal/port/oapi/cfg.yaml`
- Create: `internal/port/oapi/strict.cfg.yaml`
- Create: `internal/port/oapi/gen.go`

- [ ] **Step 1: Create the types generation config**

Create `internal/port/oapi/cfg.yaml`:

```yaml
package: oapi
generate:
  models: true
output: oapi_types.gen.go
```

- [ ] **Step 2: Create the strict server generation config**

Create `internal/port/oapi/strict.cfg.yaml`:

```yaml
package: oapi
generate:
  gorilla-server: true
  strict-server: true
  embedded-spec: true
output: oapi_server.gen.go
```

Note: `gorilla-server: true` is required by oapi-codegen to generate the strict server handler wiring. The `StrictServerInterface` is router-agnostic. `embedded-spec: true` embeds the OpenAPI JSON for serving via `/openapi.json` endpoint later.

- [ ] **Step 3: Create the generate directive file**

Create `internal/port/oapi/gen.go`:

```go
package oapi

//go:generate go tool oapi-codegen -config cfg.yaml ../../openapi.json
//go:generate go tool oapi-codegen -config strict.cfg.yaml ../../openapi.json
```

- [ ] **Step 4: Commit with caveman-commit**

```
chore(oapi): add oapi-codegen configs and generate directive
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

### Task 3: Generate Code and Verify Output

**Files:**
- Generated: `internal/port/oapi/oapi_types.gen.go`
- Generated: `internal/port/oapi/oapi_server.gen.go`

- [ ] **Step 1: Run go generate**

Run:
```bash
go generate ./internal/port/oapi/
```

Expected: Two files created — `oapi_types.gen.go` and `oapi_server.gen.go`. No errors.

- [ ] **Step 2: Verify generated code compiles**

Run:
```bash
go build ./...
```

Expected: Success (no errors)

- [ ] **Step 3: Verify StrictServerInterface methods exist**

Run:
```bash
grep -n 'func.*StrictServerInterface' internal/port/oapi/oapi_server.gen.go | head -20
```

Expected output should include method signatures for:
- `CreateNote`
- `ListNotes`
- `GetNote`
- `DeleteNote`
- `CreateLogEntry`
- `ListLogEntries`

The interface name will be `StrictServerInterface` with a `context.Context` parameter and request/response objects per operation.

- [ ] **Step 4: Verify generated types exist**

Run:
```bash
grep -E '^type (Note|NoteList|CreateNoteRequest|PomodoroState|Error|LogAction|CreateLogEntryRequest|LogEntry|LogEntryList) ' internal/port/oapi/oapi_types.gen.go
```

Expected: All eight types found:
```
type CreateNoteRequest struct {
type CreateLogEntryRequest struct {
type Error struct {
type LogAction string
type LogEntry struct {
type LogEntryList struct {
type Note struct {
type NoteList struct {
type PomodoroState string
```

- [ ] **Step 5: Commit with caveman-commit**

```
feat(oapi): generate strict server interface and types
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

### Task 4: Create Translation Layer — PomodoroState and Session Conversions

**Files:**
- Create: `internal/port/oapi/http/translate.go`

The translation layer converts between generated OAPI types (string enums) and domain types (int enums). Generated `PomodoroState` is a string type with values `"work"`, `"short_break"`, `"long_break"`. Domain `pomodoro.State` is an `int` iota with values `Work=0`, `ShortBreak=1`, `LongBreak=2`. Generated `LogAction` maps to domain `log.Action`, and generated `PomodoroState` also maps to domain `log.Session`.

- [ ] **Step 1: Create the translate.go file**

Create `internal/port/oapi/http/translate.go`:

```go
package http

import (
	"fmt"
	"time"

	oapi "github.com/4sp1/vipo-go/internal/port/oapi"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func PomodoroStateToDomain(s oapi.PomodoroState) (pomodoro.State, error) {
	switch s {
	case oapi.Work:
		return pomodoro.Work, nil
	case oapi.ShortBreak:
		return pomodoro.ShortBreak, nil
	case oapi.LongBreak:
		return pomodoro.LongBreak, nil
	default:
		return pomodoro.State(0), fmt.Errorf("unknown pomodoro state: %s", s)
	}
}

func PomodoroStateFromDomain(s pomodoro.State) (oapi.PomodoroState, error) {
	switch s {
	case pomodoro.Work:
		return oapi.Work, nil
	case pomodoro.ShortBreak:
		return oapi.ShortBreak, nil
	case pomodoro.LongBreak:
		return oapi.LongBreak, nil
	default:
		return "", fmt.Errorf("unknown pomodoro state: %d", s)
	}
}

func LogActionToDomain(a oapi.LogAction) (log.Action, error) {
	switch a {
	case oapi.Start:
		return log.ActionStart, nil
	case oapi.Pause:
		return log.ActionPause, nil
	case oapi.Reset:
		return log.ActionReset, nil
	case oapi.Expire:
		return log.ActionExpire, nil
	case oapi.Resume:
		return log.ActionResume, nil
	case oapi.Select:
		return log.ActionSelect, nil
	default:
		return log.ActionUnknown, fmt.Errorf("unknown log action: %s", a)
	}
}

func LogActionFromDomain(a log.Action) (oapi.LogAction, error) {
	switch a {
	case log.ActionStart:
		return oapi.Start, nil
	case log.ActionPause:
		return oapi.Pause, nil
	case log.ActionReset:
		return oapi.Reset, nil
	case log.ActionExpire:
		return oapi.Expire, nil
	case log.ActionResume:
		return oapi.Resume, nil
	case log.ActionSelect:
		return oapi.Select, nil
	default:
		return "", fmt.Errorf("unknown log action: %d", a)
	}
}

func SessionToDomain(s oapi.PomodoroState) (log.Session, error) {
	switch s {
	case oapi.Work:
		return log.SessionWork, nil
	case oapi.ShortBreak:
		return log.SessionShortBreak, nil
	case oapi.LongBreak:
		return log.SessionLongBreak, nil
	default:
		return log.SessionUnknown, fmt.Errorf("unknown session: %s", s)
	}
}

func SessionFromDomain(s log.Session) (oapi.PomodoroState, error) {
	switch s {
	case log.SessionWork:
		return oapi.Work, nil
	case log.SessionShortBreak:
		return oapi.ShortBreak, nil
	case log.SessionLongBreak:
		return oapi.LongBreak, nil
	default:
		return "", fmt.Errorf("unknown session: %d", s)
	}
}

func TimeToRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func TimeFromRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
```

- [ ] **Step 2: Verify translation code compiles**

Run:
```bash
go build ./internal/port/oapi/http/
```

Expected: Success (no errors)

- [ ] **Step 3: Commit with caveman-commit**

```
feat(oapi/http): add domain ↔ oapi type translations

PomodoroState, LogAction, Session string↔int enums.
RFC3339 ↔ time.Time helpers.
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

### Task 5: Write Tests for Translation Layer

**Files:**
- Create: `internal/port/oapi/http/translate_test.go`

- [ ] **Step 1: Write the failing test for PomodoroState conversions**

Create `internal/port/oapi/http/translate_test.go`:

```go
package http

import (
	"testing"
	"time"

	oapi "github.com/4sp1/vipo-go/internal/port/oapi"

	"github.com/4sp1/vipo-go/internal/domain/state/log"
	"github.com/4sp1/vipo-go/internal/domain/state/pomodoro"
)

func TestPomodoroStateToDomain(t *testing.T) {
	tests := []struct {
		input    oapi.PomodoroState
		expected pomodoro.State
	}{
		{oapi.Work, pomodoro.Work},
		{oapi.ShortBreak, pomodoro.ShortBreak},
		{oapi.LongBreak, pomodoro.LongBreak},
	}
	for _, tt := range tests {
		got, err := PomodoroStateToDomain(tt.input)
		if err != nil {
			t.Errorf("PomodoroStateToDomain(%q): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("PomodoroStateToDomain(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestPomodoroStateToDomain_UnknownValue(t *testing.T) {
	_, err := PomodoroStateToDomain("unknown")
	if err == nil {
		t.Error("expected error for unknown PomodoroState, got nil")
	}
}

func TestPomodoroStateFromDomain(t *testing.T) {
	tests := []struct {
		input    pomodoro.State
		expected oapi.PomodoroState
	}{
		{pomodoro.Work, oapi.Work},
		{pomodoro.ShortBreak, oapi.ShortBreak},
		{pomodoro.LongBreak, oapi.LongBreak},
	}
	for _, tt := range tests {
		got, err := PomodoroStateFromDomain(tt.input)
		if err != nil {
			t.Errorf("PomodoroStateFromDomain(%d): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("PomodoroStateFromDomain(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestPomodoroStateFromDomain_UnknownValue(t *testing.T) {
	_, err := PomodoroStateFromDomain(pomodoro.State(99))
	if err == nil {
		t.Error("expected error for unknown pomodoro.State, got nil")
	}
}

func TestLogActionToDomain(t *testing.T) {
	tests := []struct {
		input    oapi.LogAction
		expected log.Action
	}{
		{oapi.Start, log.ActionStart},
		{oapi.Pause, log.ActionPause},
		{oapi.Reset, log.ActionReset},
		{oapi.Expire, log.ActionExpire},
		{oapi.Resume, log.ActionResume},
		{oapi.Select, log.ActionSelect},
	}
	for _, tt := range tests {
		got, err := LogActionToDomain(tt.input)
		if err != nil {
			t.Errorf("LogActionToDomain(%q): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("LogActionToDomain(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestLogActionToDomain_UnknownValue(t *testing.T) {
	_, err := LogActionToDomain("unknown")
	if err == nil {
		t.Error("expected error for unknown LogAction, got nil")
	}
}

func TestLogActionFromDomain(t *testing.T) {
	tests := []struct {
		input    log.Action
		expected oapi.LogAction
	}{
		{log.ActionStart, oapi.Start},
		{log.ActionPause, oapi.Pause},
		{log.ActionReset, oapi.Reset},
		{log.ActionExpire, oapi.Expire},
		{log.ActionResume, oapi.Resume},
		{log.ActionSelect, oapi.Select},
	}
	for _, tt := range tests {
		got, err := LogActionFromDomain(tt.input)
		if err != nil {
			t.Errorf("LogActionFromDomain(%d): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("LogActionFromDomain(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLogActionFromDomain_UnknownValue(t *testing.T) {
	_, err := LogActionFromDomain(log.ActionUnknown)
	if err == nil {
		t.Error("expected error for ActionUnknown, got nil")
	}
}

func TestSessionToDomain(t *testing.T) {
	tests := []struct {
		input    oapi.PomodoroState
		expected log.Session
	}{
		{oapi.Work, log.SessionWork},
		{oapi.ShortBreak, log.SessionShortBreak},
		{oapi.LongBreak, log.SessionLongBreak},
	}
	for _, tt := range tests {
		got, err := SessionToDomain(tt.input)
		if err != nil {
			t.Errorf("SessionToDomain(%q): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("SessionToDomain(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestSessionToDomain_UnknownValue(t *testing.T) {
	_, err := SessionToDomain("unknown")
	if err == nil {
		t.Error("expected error for unknown session, got nil")
	}
}

func TestSessionFromDomain(t *testing.T) {
	tests := []struct {
		input    log.Session
		expected oapi.PomodoroState
	}{
		{log.SessionWork, oapi.Work},
		{log.SessionShortBreak, oapi.ShortBreak},
		{log.SessionLongBreak, oapi.LongBreak},
	}
	for _, tt := range tests {
		got, err := SessionFromDomain(tt.input)
		if err != nil {
			t.Errorf("SessionFromDomain(%d): unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("SessionFromDomain(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSessionFromDomain_UnknownValue(t *testing.T) {
	_, err := SessionFromDomain(log.SessionUnknown)
	if err == nil {
		t.Error("expected error for SessionUnknown, got nil")
	}
}

func TestTimeToRFC3339(t *testing.T) {
	ts := time.Date(2026, 6, 14, 12, 30, 0, 0, time.UTC)
	got := TimeToRFC3339(ts)
	expected := "2026-06-14T12:30:00Z"
	if got != expected {
		t.Errorf("TimeToRFC3339() = %q, want %q", got, expected)
	}
}

func TestTimeToRFC3339_NormalizesToUTC(t *testing.T) {
	loc := time.FixedZone("EST", -5*3600)
	ts := time.Date(2026, 6, 14, 7, 30, 0, 0, loc)
	got := TimeToRFC3339(ts)
	expected := "2026-06-14T12:30:00Z"
	if got != expected {
		t.Errorf("TimeToRFC3339() = %q, want %q", got, expected)
	}
}

func TestTimeFromRFC3339(t *testing.T) {
	input := "2026-06-14T12:30:00Z"
	got, err := TimeFromRFC3339(input)
	if err != nil {
		t.Errorf("TimeFromRFC3339(%q): unexpected error: %v", input, err)
	}
	expected := time.Date(2026, 6, 14, 12, 30, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("TimeFromRFC3339(%q) = %v, want %v", input, got, expected)
	}
}

func TestTimeFromRFC3339_InvalidInput(t *testing.T) {
	_, err := TimeFromRFC3339("not-a-timestamp")
	if err == nil {
		t.Error("expected error for invalid RFC3339 input, got nil")
	}
}

func TestPomodoroStateRoundTrip(t *testing.T) {
	states := []pomodoro.State{pomodoro.Work, pomodoro.ShortBreak, pomodoro.LongBreak}
	for _, original := range states {
		oapiState, err := PomodoroStateFromDomain(original)
		if err != nil {
			t.Errorf("PomodoroStateFromDomain(%d): unexpected error: %v", original, err)
		}
		back, err := PomodoroStateToDomain(oapiState)
		if err != nil {
			t.Errorf("PomodoroStateToDomain(%q): unexpected error: %v", oapiState, err)
		}
		if back != original {
			t.Errorf("round trip: domain %d → oapi %q → domain %d", original, oapiState, back)
		}
	}
}

func TestLogActionRoundTrip(t *testing.T) {
	actions := []log.Action{log.ActionStart, log.ActionPause, log.ActionReset, log.ActionExpire, log.ActionResume, log.ActionSelect}
	for _, original := range actions {
		oapiAction, err := LogActionFromDomain(original)
		if err != nil {
			t.Errorf("LogActionFromDomain(%d): unexpected error: %v", original, err)
		}
		back, err := LogActionToDomain(oapiAction)
		if err != nil {
			t.Errorf("LogActionToDomain(%q): unexpected error: %v", oapiAction, err)
		}
		if back != original {
			t.Errorf("round trip: domain %d → oapi %q → domain %d", original, oapiAction, back)
		}
	}
}

func TestTimeRoundTrip(t *testing.T) {
	original := time.Date(2026, 6, 14, 12, 30, 0, 0, time.UTC)
	rfc3339 := TimeToRFC3339(original)
	back, err := TimeFromRFC3339(rfc3339)
	if err != nil {
		t.Errorf("TimeFromRFC3339(%q): unexpected error: %v", rfc3339, err)
	}
	original = original.Truncate(time.Second)
	if !back.Equal(original) {
		t.Errorf("round trip: %v → %q → %v", original, rfc3339, back)
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run:
```bash
go test ./internal/port/oapi/http/ -v
```

Expected: All tests PASS.

- [ ] **Step 3: Commit with caveman-commit**

```
test(oapi/http): add translation layer tests

Covers PomodoroState, LogAction, Session enums and
RFC3339 ↔ time.Time with round-trip and error cases.
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

### Task 6: Final Verification and AGENTS.md Update

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Run go vet on oapi package**

Run:
```bash
go vet ./internal/port/oapi/...
```

Expected: No issues reported.

- [ ] **Step 2: Run all tests**

Run:
```bash
go test ./...
```

Expected: All tests PASS.

- [ ] **Step 3: Update AGENTS.md — add oapi-codegen section**

Add the following to `AGENTS.md` after the `## Gotchas` section:

```markdown

## OAPI Code Generation

- **Generator:** oapi-codegen v2 (managed via `go tool`)
- **Config files:** `internal/port/oapi/cfg.yaml` (types), `internal/port/oapi/strict.cfg.yaml` (strict server + gorilla handler)
- **Generation command:** `go generate ./internal/port/oapi/`
- **Generated files:** `oapi_types.gen.go`, `oapi_server.gen.go` — do NOT edit these; regenerate instead
- **StrictServerInterface:** Router-agnostic Go interface. Service layer implements this to provide typed HTTP handlers.
- **Translation layer:** `internal/port/oapi/http/translate.go` converts between OAPI string enums and domain int enums.
- **Important:** Domain types use int iota (`pomodoro.State`, `log.Action`, `log.Session`). OAPI types use string enums (`PomodoroState`, `LogAction`). The translation layer bridges this gap.
```

- [ ] **Step 4: Commit with caveman-commit**

```
docs(agents): add oapi-codegen generation docs
```

Use `jj describe` with temp file pattern from core-commands skill, then `jj new`.

---

## Success Criteria

- [ ] `oapi-codegen` config files exist (`internal/port/oapi/cfg.yaml` and `internal/port/oapi/strict.cfg.yaml`)
- [ ] `go generate ./internal/port/oapi/` produces compilable Go code
- [ ] Generated `StrictServerInterface` includes: `CreateNote`, `ListNotes`, `GetNote`, `DeleteNote`
- [ ] Generated types include: `Note`, `NoteList`, `CreateNoteRequest`, `PomodoroState`, `Error`
- [ ] Translation functions exist in `internal/port/oapi/http/`: generated `PomodoroState` ↔ `pomodoro.State`, RFC3339 ↔ `time.Time`
- [ ] `go vet ./internal/port/oapi` passes
- [ ] All translation tests pass

## Notes

- `gorilla-server: true` in `strict.cfg.yaml` is required by oapi-codegen to generate the `StrictServerInterface`. The interface itself is router-agnostic. When routing is wired up (separate issue), the gorilla handler code will be used.
- The OpenAPI spec includes log endpoints (`CreateLogEntry`, `ListLogEntries`) and their types (`LogAction`, `CreateLogEntryRequest`, `LogEntry`, `LogEntryList`). These are generated alongside the note types — they'll be used in a future issue for log endpoint wiring.
- `ActionUnknown` and `ActionNewNote` in the domain `log.Action` enum are not in the OpenAPI `LogAction` enum. The `LogActionFromDomain` function returns an error for these values since they have no OAPI equivalent.
- Domain `log.Session` maps to the same OAPI `PomodoroState` string values as domain `pomodoro.State`. The translation layer has separate functions for each domain type.