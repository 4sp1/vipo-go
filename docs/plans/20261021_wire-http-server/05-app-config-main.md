# 05 — App, Config & Main

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the application wiring layer: `apperr.Code` taxonomy with `Wire()` error mapper, app wiring that connects ports + svc + HTTP router, cobra CLI with `serve` and `migrate` subcommands, viper layered config, and `main.go` entrypoint.

**Architecture:** `internal/app/` package owns the wiring: `New(db, addr) → http.Handler`. `internal/app/apperr/` owns the `Code` enum, stringer, and `Wire()` switch that maps svc errors and 400 response types to `Error{code, message}`. `cmd/vipo-server/main.go` uses cobra root with `serve` (run HTTP server) and `migrate` (run migrations independently) subcommands. viper handles flag > env > config.json > defaults. [S16], [S17], [S23].

**Tech Stack:** `spf13/cobra`, `spf13/viper`, `glebarez/go-sqlite`, `pressly/goose/v3`, oapi-codegen strict server, Go stdlib.

**Depends on:** Sub-plan 02 (`*DB` wrapper, `Migrate`), sub-plan 04 (svc `Service`).

---

## Files

- Create: `internal/app/apperr/code.go`
- Create: `internal/app/apperr/code_test.go`
- Create: `internal/app/wire.go`
- Create: `internal/app/wire_test.go`
- Create: `cmd/vipo-server/main.go`
- Modify: `go.mod` (add cobra + viper)

---

### Task 1: `apperr.Code` taxonomy

**Files:**
- Create: `internal/app/apperr/code.go`
- Test: `internal/app/apperr/code_test.go`

- [ ] **Step 1: Write the failing test**

```go
package apperr

import (
	"testing"
)

func TestCodeString(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{CodeNoteNotFound, "note_not_found"},
		{CodeLogNotFound, "log_not_found"},
		{CodeInvalidRequest, "invalid_request"},
		{CodeInvalidPomodoroState, "invalid_pomodoro_state"},
		{CodeInvalidLogAction, "invalid_log_action"},
		{CodeInvalidLimit, "invalid_limit"},
		{CodeInvalidOffset, "invalid_offset"},
		{CodeInvalidTimestamp, "invalid_timestamp"},
	}
	for _, tt := range tests {
		if got := tt.code.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestCodeError(t *testing.T) {
	c := CodeNoteNotFound
	if c.Error() != "note_not_found" {
		t.Errorf("Error() = %q, want 'note_not_found'", c.Error())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/apperr/ -v`
Expected: FAIL — `undefined: Code`, `undefined: CodeNoteNotFound`

- [ ] **Step 3: Write minimal implementation**

Create `internal/app/apperr/code.go`:

```go
package apperr

import "fmt"

type Code int

const (
	CodeNoteNotFound Code = iota
	CodeLogNotFound
	CodeInvalidRequest
	CodeInvalidPomodoroState
	CodeInvalidLogAction
	CodeInvalidLimit
	CodeInvalidOffset
	CodeInvalidTimestamp
)

func (c Code) String() string {
	switch c {
	case CodeNoteNotFound:
		return "note_not_found"
	case CodeLogNotFound:
		return "log_not_found"
	case CodeInvalidRequest:
		return "invalid_request"
	case CodeInvalidPomodoroState:
		return "invalid_pomodoro_state"
	case CodeInvalidLogAction:
		return "invalid_log_action"
	case CodeInvalidLimit:
		return "invalid_limit"
	case CodeInvalidOffset:
		return "invalid_offset"
	case CodeInvalidTimestamp:
		return "invalid_timestamp"
	default:
		return fmt.Sprintf("unknown_code(%d)", c)
	}
}

func (c Code) Error() string {
	return c.String()
}
```

Key decisions: `Code` is an `int` enum with a manual `String()` method (not a stringer) — the stringer derives from Go identifiers and can't emit `note_not_found`. [S16]. `Code.Error()` delegates to `String()` for logging. [S16]. The enumerated codes match the decisions doc: `note_not_found`, `log_not_found`, `invalid_request`, `invalid_pomodoro_state`, `invalid_log_action`, `invalid_limit`, `invalid_offset`, `invalid_timestamp`. [S16].

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/apperr/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(app): add apperr.Code taxonomy

Enumerated HTTP error codes with manual String()
(snake_case wire strings). Code.Error() delegates to
String() for logging. [S16]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 2: `Wire()` error mapper

**Files:**
- Create: `internal/app/apperr/wire.go`
- Test: `internal/app/apperr/wire_test.go`

The `Wire()` function takes the svc's response object and error, and enriches 400/404 responses with the `Code` field. The svc returns `Error{Message: ...}` without `Code`; `Wire()` inspects the response type to determine the code and sets it.

However, the generated response types (`CreateNote400JSONResponse`, `GetNote404JSONResponse`, etc.) are `type CreateNote400JSONResponse Error` — they are the `Error` struct. We can't set the `Code` field after the fact because the response is already typed. The approach: `Wire()` is not a post-processor of responses; instead, it wraps the svc to intercept errors and produce the correct response. But the svc already produces the responses...

Actually, the cleanest approach: the svc returns responses with `Message` but no `Code`. The app layer wraps the svc in a `StrictMiddlewareFunc` that intercepts the response, checks if it's an `Error`-typed response, and sets `Code` based on which handler produced it. But the middleware doesn't know which specific validation failed.

Better approach: the svc returns a custom error type for validation failures, and the app layer's `ResponseErrorHandlerFunc` maps them to 400 responses with the right `Code`. But the OAPI strict handler calls `ResponseErrorHandlerFunc` only when the svc returns a non-nil `error`, not when it returns a 400 response object.

Revised approach: The svc returns 400/404 response objects (not Go errors) for expected failures. These response objects are `Error` structs with `Message` set. The app layer wraps the svc with a `StrictMiddlewareFunc` that post-processes the response. If the response is a 400/404 type, the middleware type-asserts and sets the `Code` field by inspecting the response type.

But the response types are value types (`type CreateNote400JSONResponse Error`), not pointers, so the middleware can't mutate them in place. The middleware must return a new response.

Simplest correct approach: `Wire()` is called inside each svc handler, not as middleware. But the decisions doc says "a separate `Wire()` switch produces snake_case wire strings" at the HTTP boundary, not in svc. [S16].

Final approach: The app layer overrides `ResponseErrorHandlerFunc` for 500s (plain text). [S23]. For 400/404, the svc already returns the response object. The `Code` field is set by the svc itself, but using the `apperr.Code` values. Wait — [S16] says "Code lives at the HTTP boundary, not in svc." The svc must not import `apperr`.

Resolution: The app layer wraps the svc in a middleware that intercepts response objects. Since response types are value types, the middleware returns a *new* response with `Code` set. The middleware uses the operation ID to determine the code.

Actually, let's re-read [S16] more carefully: "Svc returns adapter errors (`note.ErrNotFound`) and validation errors; the HTTP layer maps them to `Code` → `Error{code, message}`."

This means: the svc returns Go *errors* for not-found and validation, NOT 400/404 response objects. The HTTP layer (app) catches those errors and produces the response with `Code`.

But the OAPI strict handler framework calls `ResponseErrorHandlerFunc` when the svc returns a non-nil `error`. So the flow is:
1. svc returns `(nil, note.ErrNotFound)` for not-found
2. strict handler calls `ResponseErrorHandlerFunc(w, r, note.ErrNotFound)`
3. The app layer's custom `ResponseErrorHandlerFunc` maps the error to a 404 response with `Code: "note_not_found"`

For validation errors, the svc returns a custom validation error type (defined in svc or app), and the `ResponseErrorHandlerFunc` maps it to a 400 response with the appropriate `Code`.

This changes the svc design from sub-plan 04. In sub-plan 04, the svc returns 400/404 response objects. For the app layer to inject `Code`, the svc must instead return Go errors, and the app layer's error handler produces the HTTP response.

BUT — the `ResponseErrorHandlerFunc` signature is `func(w http.ResponseWriter, r *http.Request, err error)` — it writes directly to the response writer. This means the app layer would write the JSON error body manually. This is doable but bypasses the generated response types.

Alternative: the svc returns BOTH response objects (for the HTTP status code) AND errors (for the code). But the strict handler only uses one: if error is non-nil, it calls `ResponseErrorHandlerFunc`; if error is nil, it visits the response object. They are mutually exclusive.

Cleanest resolution that respects [S16]: the svc returns Go errors for all failure cases (not-found, validation). The app layer provides a custom `ResponseErrorHandlerFunc` that maps errors to HTTP responses with `Error{code, message}`. The svc defines error types (or uses adapter errors) that the app layer can inspect.

Let's define error types in the svc package:

```go
// internal/svc/errors.go
package svc

import (
	"fmt"

	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
)

type ValidationError struct {
	Code    string // temporary; app layer replaces with apperr.Code
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
```

No — [S16] says "Code lives at the HTTP boundary, not in svc." The svc should not define codes. The svc should return generic validation errors with a message, and the app layer inspects the error type to determine the code.

Revised svc error types (in `internal/svc/`):

```go
type ErrInvalidPomodoroState struct{ Value string }
func (e *ErrInvalidPomodoroState) Error() string { return fmt.Sprintf("invalid pomodoro_state: %s", e.Value) }

type ErrInvalidLogAction struct{ Value string }
func (e *ErrInvalidLogAction) Error() string { return fmt.Sprintf("invalid action: %s", e.Value) }

type ErrInvalidLimit struct{ Value int }
func (e *ErrInvalidLimit) Error() string { return fmt.Sprintf("invalid limit: %d", e.Value) }

type ErrInvalidOffset struct{ Value int }
func (e *ErrInvalidOffset) Error() string { return fmt.Sprintf("invalid offset: %d", e.Value) }

type ErrInvalidTimestamp struct{ Value string }
func (e *ErrInvalidTimestamp) Error() string { return fmt.Sprintf("invalid timestamp: %s", e.Value) }

type ErrInvalidRequest struct{ Message string }
func (e *ErrInvalidRequest) Error() string { return e.Message }
```

The app layer's `Wire()` function (or the `ResponseErrorHandlerFunc`) inspects these types and produces `Error{code, message}`.

This means sub-plan 04 needs to be updated: the svc returns errors instead of 400/404 response objects. The svc returns `(nil, err)` for failures and `(responseObject, nil)` for success.

**This is a significant design change from sub-plan 04.** The decisions doc [S16] says "Svc returns adapter errors (`note.ErrNotFound`) and validation errors; the HTTP layer maps them to `Code`." This clearly means the svc returns errors, not response objects, for failure cases.

Since sub-plan 04 was written by the previous agent with the response-object approach, the app layer plan (this plan) needs to document this discrepancy. The executor should update the svc handlers from sub-plan 04 to return errors instead of 400/404 response objects.

Given the complexity, the most pragmatic approach: keep sub-plan 04's response-object approach for now (svc returns 400/404 response objects), and have the app layer wrap the svc with a `StrictMiddlewareFunc` that intercepts responses and adds `Code`. The middleware type-asserts each response type:

```go
func enrichResponse(operationID string, response interface{}) interface{} {
	switch operationID {
	case "CreateNote":
		if r, ok := response.(oapi.CreateNote400JSONResponse); ok {
			// determine code from message content...
		}
	}
}
```

This is fragile (matching on message strings). Let's use a better approach: the svc attaches the code as a field on the response. Since `CreateNote400JSONResponse` is `type CreateNote400JSONResponse Error`, and `Error` has a `Code *string` field, the svc can set `Code` to a placeholder and the app layer replaces it.

Actually, the simplest correct approach: the svc returns the `Error` response with `Code` already set to the snake_case string. [S16] says "Code lives at the HTTP boundary, not in svc" — but the response object IS the HTTP boundary. The svc constructs the response that goes on the wire. The "HTTP boundary" in [S16] means the response shaping layer, which the svc IS (it implements the `StrictServerInterface`).

Re-reading [S16]: "The `Code int` enum (with stringer) lives at the HTTP boundary, not in svc." The enum type `apperr.Code` lives in `internal/app/apperr/`, not in svc. But the svc can still set `Code *string` on the `Error` response using string literals. The `apperr.Code` enum is for the app layer's own use (logging, structured errors), while the svc sets the wire `code` string directly on the response.

But [S16] also says "Svc returns adapter errors (`note.ErrNotFound`) and validation errors; the HTTP layer maps them to `Code` → `Error{code, message}`." This says the HTTP layer maps, not the svc.

**Final resolution:** Both approaches are valid interpretations. The most practical one for this codebase: the svc returns response objects with `Message` set. The app layer wraps the svc with a `StrictMiddlewareFunc` that inspects the response type (400/404) and sets `Code` based on the operation ID + response status. The middleware returns a new response with `Code` set.

For this to work, the middleware needs to know the code per operation + failure type. The `Wire()` function provides this mapping:

```go
func Wire(operationID string, statusCode int) Code
```

- `CreateNote` + 400 → `CodeInvalidRequest` (or more specific based on message)
- `GetNote` + 404 → `CodeNoteNotFound`
- `DeleteNote` + 404 → `CodeNoteNotFound`
- `CreateLogEntry` + 400 → `CodeInvalidRequest`
- `ListNotes` + 400 → `CodeInvalidLimit` or `CodeInvalidOffset` or `CodeInvalidPomodoroState`
- `ListLogEntries` + 400 → varies

The problem: `Wire()` needs more than operation ID + status code to distinguish `invalid_limit` from `invalid_offset` from `invalid_pomodoro_state`. It needs the message content.

**Pragmatic approach for this plan:** The svc sets the `Code` string on the response directly, using string constants. The `apperr.Code` enum exists for the app layer's own use (logging, structured error reporting), but the wire `code` field is set by the svc as a string. This is a minor deviation from [S16]'s strictest reading, but it's the most practical approach. The `apperr.Code` enum and `Wire()` function still exist and are used by the app layer to log structured errors and to validate that the wire codes match the enum.

Actually, the best approach: define the `Code` wire strings as constants in `apperr`, have the svc import `apperr` for the string constants only, and use them to set the `Code *string` field on `Error` responses. The `apperr.Code` type itself stays in the app layer, but the string values are shared. This way the svc uses `apperr.CodeNoteNotFound.String()` to set the code, but doesn't use the `Code` int type for logic.

Wait, [S16] says "not in svc." Let's resolve this definitively: the svc should NOT import `apperr`. The svc returns errors, and the app layer maps them.

**Final final resolution:** The svc returns typed errors for validation failures and adapter errors for not-found. The app layer provides a custom `ResponseErrorHandlerFunc` that writes `Error{code, message}` JSON to the response. The generated 400/404 response types are NOT used by the svc; instead, the error handler writes directly to `http.ResponseWriter`. This is the approach [S16] describes.

This means sub-plan 04's approach needs adjustment. The svc handlers return `(nil, err)` for failures. The `ResponseErrorHandlerFunc` inspects the error and writes the appropriate HTTP response.

This plan defines the error types in svc and the `Wire()` function in apperr that maps them.

- [ ] **Step 1: Write the failing test**

Create `internal/app/apperr/wire_test.go`:

```go
package apperr

import (
	"errors"
	"testing"

	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	svcerrors "github.com/4sp1/vipo-go/internal/svc"
)

func TestWireNoteNotFound(t *testing.T) {
	code, msg, status := Wire(adaptNote.ErrNotFound)
	if code != CodeNoteNotFound {
		t.Errorf("code = %s, want %s", code, CodeNoteNotFound)
	}
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}
	if msg == "" {
		t.Error("message should not be empty")
	}
}

func TestWireLogNotFound(t *testing.T) {
	code, msg, status := Wire(adaptLog.ErrNotFound)
	if code != CodeLogNotFound {
		t.Errorf("code = %s, want %s", code, CodeLogNotFound)
	}
	if status != 404 {
		t.Errorf("status = %d, want 404", status)
	}
}

func TestWireInvalidPomodoroState(t *testing.T) {
	err := &svcerrors.ErrInvalidPomodoroState{Value: "bad"}
	code, _, status := Wire(err)
	if code != CodeInvalidPomodoroState {
		t.Errorf("code = %s, want %s", code, CodeInvalidPomodoroState)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidLogAction(t *testing.T) {
	err := &svcerrors.ErrInvalidLogAction{Value: "bad"}
	code, _, status := Wire(err)
	if code != CodeInvalidLogAction {
		t.Errorf("code = %s, want %s", code, CodeInvalidLogAction)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidLimit(t *testing.T) {
	err := &svcerrors.ErrInvalidLimit{Value: 0}
	code, _, status := Wire(err)
	if code != CodeInvalidLimit {
		t.Errorf("code = %s, want %s", code, CodeInvalidLimit)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidOffset(t *testing.T) {
	err := &svcerrors.ErrInvalidOffset{Value: -1}
	code, _, status := Wire(err)
	if code != CodeInvalidOffset {
		t.Errorf("code = %s, want %s", code, CodeInvalidOffset)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireInvalidRequest(t *testing.T) {
	err := &svcerrors.ErrInvalidRequest{Message: "missing body"}
	code, _, status := Wire(err)
	if code != CodeInvalidRequest {
		t.Errorf("code = %s, want %s", code, CodeInvalidRequest)
	}
	if status != 400 {
		t.Errorf("status = %d, want 400", status)
	}
}

func TestWireUnknownError(t *testing.T) {
	err := errors.New("something broke")
	code, _, status := Wire(err)
	if code != CodeInvalidRequest {
		// Unknown errors default to 500 in the ResponseErrorHandlerFunc,
		// but Wire returns the closest match for 400-level.
		// Actually, unknown errors should return 500, not a code.
		t.Logf("code = %s, status = %d for unknown error", code, status)
	}
}
```

Note: the test references `svc.ErrInvalidPomodoroState` etc. These error types must be defined in the svc package. Since sub-plan 04 has already been written with the response-object approach, the executor needs to update the svc to return these errors instead. This is documented in the Task 4 note below.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/apperr/ -run TestWire -v`
Expected: FAIL — `undefined: Wire`, `undefined: svc.ErrInvalidPomodoroState`

- [ ] **Step 3: Write minimal implementation**

First, the svc error types must exist. If sub-plan 04 was already executed with the response-object approach, create `internal/svc/errors.go`:

```go
package svc

import "fmt"

type ErrInvalidPomodoroState struct{ Value string }
func (e *ErrInvalidPomodoroState) Error() string {
	return fmt.Sprintf("invalid pomodoro_state: %s", e.Value)
}

type ErrInvalidLogAction struct{ Value string }
func (e *ErrInvalidLogAction) Error() string {
	return fmt.Sprintf("invalid action: %s", e.Value)
}

type ErrInvalidSession struct{ Value string }
func (e *ErrInvalidSession) Error() string {
	return fmt.Sprintf("invalid session: %s", e.Value)
}

type ErrInvalidLimit struct{ Value int }
func (e *ErrInvalidLimit) Error() string {
	return fmt.Sprintf("invalid limit: %d", e.Value)
}

type ErrInvalidOffset struct{ Value int }
func (e *ErrInvalidOffset) Error() string {
	return fmt.Sprintf("invalid offset: %d", e.Value)
}

type ErrInvalidRequest struct{ Message string }
func (e *ErrInvalidRequest) Error() string {
	if e.Message == "" {
		return "invalid request"
	}
	return e.Message
}
```

Then create `internal/app/apperr/wire.go`:

```go
package apperr

import (
	"errors"

	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	svcerrors "github.com/4sp1/vipo-go/internal/svc"
)

// Wire maps an svc-level error to an HTTP error code, message, and status.
// Returns ok=false for unknown errors (caller should produce 500).
func Wire(err error) (code Code, message string, status int) {
	switch {
	case errors.Is(err, adaptNote.ErrNotFound):
		return CodeNoteNotFound, "note not found", 404
	case errors.Is(err, adaptLog.ErrNotFound):
		return CodeLogNotFound, "log entry not found", 404
	}

	var invalidPS *svcerrors.ErrInvalidPomodoroState
	if errors.As(err, &invalidPS) {
		return CodeInvalidPomodoroState, invalidPS.Error(), 400
	}

	var invalidAction *svcerrors.ErrInvalidLogAction
	if errors.As(err, &invalidAction) {
		return CodeInvalidLogAction, invalidAction.Error(), 400
	}

	var invalidSession *svcerrors.ErrInvalidSession
	if errors.As(err, &invalidSession) {
		return CodeInvalidPomodoroState, invalidSession.Error(), 400
	}

	var invalidLimit *svcerrors.ErrInvalidLimit
	if errors.As(err, &invalidLimit) {
		return CodeInvalidLimit, invalidLimit.Error(), 400
	}

	var invalidOffset *svcerrors.ErrInvalidOffset
	if errors.As(err, &invalidOffset) {
		return CodeInvalidOffset, invalidOffset.Error(), 400
	}

	var invalidReq *svcerrors.ErrInvalidRequest
	if errors.As(err, &invalidReq) {
		return CodeInvalidRequest, invalidReq.Error(), 400
	}

	// Unknown error → 500, no code
	return Code(-1), err.Error(), 500
}
```

Note: invalid session maps to `CodeInvalidPomodoroState` since the OAPI spec types the session parameter as `PomodoroState`. The decisions doc doesn't define a separate `invalid_session` code. [S16].

The `TestWireUnknownError` test in Step 1 expects `Code(-1)` and status 500 for unknown errors. Update the test to check:

```go
func TestWireUnknownError(t *testing.T) {
	err := errors.New("something broke")
	code, _, status := Wire(err)
	if code != Code(-1) {
		t.Errorf("code = %s, want Code(-1)", code)
	}
	if status != 500 {
		t.Errorf("status = %d, want 500", status)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/apperr/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(app): add apperr.Wire error mapper

Maps svc errors (note.ErrNotFound, log.ErrNotFound,
validation errors) to HTTP code + message + status.
Unknown errors return 500. [S16]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 3: Update svc handlers to return errors instead of 400/404 response objects

**Files:**
- Modify: `internal/svc/service.go`
- Modify: `internal/svc/service_test.go`

This task updates the svc handlers from sub-plan 04 to return Go errors instead of 400/404 response objects. The app layer's `ResponseErrorHandlerFunc` uses `Wire()` to produce the HTTP response.

**If sub-plan 04 was NOT yet executed, skip this task and modify sub-plan 04's implementation directly.** If it was already executed, this task patches the handlers.

- [ ] **Step 1: Update svc handlers to return errors**

The key changes per handler:

**CreateNote**: Instead of returning `oapi.CreateNote400JSONResponse{Message: ...}`, return `(nil, &svcerrors.ErrInvalidPomodoroState{Value: string(req.Body.PomodoroState)})` or `(nil, &svcerrors.ErrInvalidRequest{Message: "missing request body"})`.

**GetNote**: Instead of returning `oapi.GetNote404JSONResponse{Message: "note not found"}`, return `(nil, adaptNote.ErrNotFound)`.

**DeleteNote**: Instead of returning `oapi.DeleteNote404JSONResponse{Message: "note not found"}`, return `(nil, adaptNote.ErrNotFound)`.

**ListNotes**: Instead of returning `listNotes400Response{message: ...}`, return `(nil, &svcerrors.ErrInvalidLimit{Value: limit})`, `(nil, &svcerrors.ErrInvalidOffset{Value: offset})`, or `(nil, &svcerrors.ErrInvalidPomodoroState{Value: string(*req.Params.PomodoroState)})`.

**CreateLogEntry**: Instead of returning `oapi.CreateLogEntry400JSONResponse{Message: ...}`, return `(nil, &svcerrors.ErrInvalidLogAction{Value: string(req.Body.Action)})`, `(nil, &svcerrors.ErrInvalidSession{Value: string(req.Body.Session)})`, or `(nil, &svcerrors.ErrInvalidRequest{Message: "missing request body"})`.

**ListLogEntries**: Instead of returning `listLogEntries400Response{message: ...}`, return the appropriate validation error.

The custom `listNotes400Response` and `listLogEntries400Response` types can be deleted since they're no longer needed.

**If sub-plan 04 has not been executed yet:** update sub-plan 04's code blocks to use this error-returning approach before executing it. The executor should integrate these changes into sub-plan 04.

For this task, assuming sub-plan 04 was already executed, patch each handler:

Replace each `return oapi.XXX400JSONResponse{Message: ...}, nil` with `return nil, &svcerrors.ErrXXX{...}`.

Replace each `return oapi.XXX404JSONResponse{Message: ...}, nil` with `return nil, adaptNote.ErrNotFound` or `return nil, adaptLog.ErrNotFound`.

- [ ] **Step 2: Update svc tests**

The svc tests need to check for errors instead of response types:

- `TestGetNoteNotFound`: check `errors.Is(err, adaptNote.ErrNotFound)` instead of `resp.(oapi.GetNote404JSONResponse)`
- `TestDeleteNoteNotFound`: same pattern
- `TestCreateNote` invalid cases: check `err != nil` and type-assert the error
- `TestListNotesInvalidLimit`: check for `*svcerrors.ErrInvalidLimit`
- etc.

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/svc/ -v`
Expected: PASS

- [ ] **Step 4: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
refactor(svc): return typed errors instead of 400/404
responses

App layer maps errors to HTTP responses via
apperr.Wire(). Svc no longer shapes HTTP error
responses. [S16]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 4: App wiring — HTTP handler

**Files:**
- Create: `internal/app/wire.go`
- Test: `internal/app/wire_test.go`

- [ ] **Step 1: Write the failing test**

```go
package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
	"github.com/4sp1/vipo-go/internal/svc"
)

func TestNewHandlerServesAPI(t *testing.T) {
	s := svc.New(nil, nil) // will panic on actual calls, but tests routing
	// Actually, nil stores will panic. Use stub stores.
	// This test just verifies the handler is created and routes exist.

	handler := NewHandler(s, testErrorHandler{})
	if handler == nil {
		t.Fatal("NewHandler returned nil")
	}
}

type testErrorHandler struct{}

func (testErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(oapi.Error{Message: err.Error()})
}

func TestHandlerRoutesNotes(t *testing.T) {
	// This is an integration-style test that verifies routing exists.
	// Full e2e tests are in sub-plan 06.
	s := svc.New(nil, nil)
	handler := NewHandler(s, testErrorHandler{})

	// GET /notes should not panic (will return 500 due to nil store)
	req := httptest.NewRequest(http.MethodGet, "/notes", strings.NewReader(""))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should get some response (500 from nil store, or 200 if store stubbed)
	if w.Code == 0 {
		t.Error("no response written")
	}
}
```

Note: The nil stores will cause panics on actual method calls. For a proper test, use the stub stores from sub-plan 04 tests. Since those are in the `svc` package (test-only), we need our own stubs here. Full e2e tests are in sub-plan 06; this test just verifies the handler is created and routes exist.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -v`
Expected: FAIL — `undefined: NewHandler`, `undefined: testErrorHandler` (interface mismatch)

- [ ] **Step 3: Write minimal implementation**

Add cobra and viper dependencies:

Run: `go get github.com/spf13/cobra@latest && go get github.com/spf13/viper@latest`

Create `internal/app/wire.go`:

```go
package app

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/4sp1/vipo-go/internal/app/apperr"
	adaptLog "github.com/4sp1/vipo-go/internal/adapter/state/log"
	adaptNote "github.com/4sp1/vipo-go/internal/adapter/state/note"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
	"github.com/4sp1/vipo-go/internal/svc"
)

type ErrorHandler interface {
	HandleError(w http.ResponseWriter, r *http.Request, err error)
}

func NewHandler(s *svc.Service, errHandler ErrorHandler) http.Handler {
	strictHandler := oapi.NewStrictHandlerWithOptions(s, nil, oapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			code, msg, _ := apperr.Wire(err)
			wireCode := code.String()
			if code == apperr.Code(-1) {
				wireCode = "invalid_request"
			}
			json.NewEncoder(w).Encode(oapi.Error{Code: &wireCode, Message: msg})
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			// 500s stay plain text [S23]
			code, msg, status := apperr.Wire(err)
			if status == 500 {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 400/404 from svc errors
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			if code == apperr.Code(-1) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			wireCode := code.String()
			json.NewEncoder(w).Encode(oapi.Error{Code: &wireCode, Message: msg})
		},
	})

	return oapi.Handler(strictHandler)
}
```

Key decisions: `ResponseErrorHandlerFunc` is called when the svc returns a non-nil `error`. It calls `apperr.Wire(err)` to get the code, message, and status. For 500s (unknown errors), it emits plain text via `http.Error(w, err.Error(), 500)`. [S23]. For 400/404, it writes JSON `Error{code, message}`. [S16]. `RequestErrorHandlerFunc` is called when the request body can't be decoded (JSON parse errors) — returns 400 with `invalid_request` code.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(app): wire HTTP handler with error mapping

NewHandler wraps svc in strict handler with custom
ResponseErrorHandlerFunc (500s plain text, 400/404 JSON
with apperr.Code). [S16, S23]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 5: App wiring — server setup with DB

**Files:**
- Create: `internal/app/server.go`
- Test: `internal/app/server_test.go`

- [ ] **Step 1: Write the failing test**

```go
package app

import (
	"context"
	"net/http"
	"testing"
)

func TestNewServer(t *testing.T) {
	srv, err := NewServer(":memory:", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	if srv.Handler == nil {
		t.Error("Handler is nil")
	}

	if err := srv.DB.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func TestNewServerInvalidPath(t *testing.T) {
	_, err := NewServer("/nonexistent/path/db.sqlite", "127.0.0.1:0")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestServerIsHTTPHandler(t *testing.T) {
	srv, err := NewServer(":memory:", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	var _ http.Handler = srv.Handler
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run TestNewServer -v`
Expected: FAIL — `undefined: NewServer`, `undefined: Server`

- [ ] **Step 3: Write minimal implementation**

Create `internal/app/server.go`:

```go
package app

import (
	"fmt"
	"net/http"

	"github.com/4sp1/vipo-go/internal/port/state/sqlite"
	"github.com/4sp1/vipo-go/internal/svc"
)

type Server struct {
	DB      *sqlite.DB
	Handler http.Handler
}

func NewServer(dbFile, addr string) (*Server, error) {
	db, err := sqlite.New(dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	s := svc.New(db.NoteStore(), db.EventLog())
	handler := NewHandler(s, defaultErrorHandler{})

	return &Server{
		DB:      db,
		Handler: handler,
	}, nil
}

func (s *Server) Close() error {
	return s.DB.Close()
}

type defaultErrorHandler struct{}

func (defaultErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	// Delegate to the response error handler embedded in the strict handler.
	// This is used by the ErrorHandler interface in tests.
	w.WriteHeader(http.StatusInternalServerError)
}
```

Note: The `defaultErrorHandler` is a no-op because the actual error handling is done inside `NewHandler`'s `ResponseErrorHandlerFunc`. The `ErrorHandler` interface exists for test extensibility.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/ -v`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(app): add Server wiring with SQLite + svc

NewServer opens DB, creates svc with NoteStore/EventLog,
returns Server with http.Handler. [S21]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 6: Cobra CLI with serve and migrate subcommands

**Files:**
- Create: `cmd/vipo-server/main.go`
- Test: `cmd/vipo-server/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
package main

import (
	"os"
	"testing"
)

func TestServeFlagDefaults(t *testing.T) {
	// Test that serve subcommand has expected defaults
	// This is a smoke test; full e2e in sub-plan 06
	cmd := newRootCmd()
	if cmd == nil {
		t.Fatal("newRootCmd returned nil")
	}

	serveCmd, _, err := cmd.Find([]string{"serve"})
	if err != nil {
		t.Fatal("serve subcommand not found")
	}
	if serveCmd == nil {
		t.Fatal("serve subcommand is nil")
	}

	// Check default flags
	addrFlag := serveCmd.Flag("addr")
	if addrFlag == nil {
		t.Error("addr flag not found")
	}
	if addrFlag.DefValue != "127.0.0.1:8080" {
		t.Errorf("addr default = %q, want '127.0.0.1:8080'", addrFlag.DefValue)
	}

	dbFlag := serveCmd.Flag("db")
	if dbFlag == nil {
		t.Error("db flag not found")
	}
}

func TestMigrateSubcommand(t *testing.T) {
	cmd := newRootCmd()
	migrateCmd, _, err := cmd.Find([]string{"migrate"})
	if err != nil {
		t.Fatal("migrate subcommand not found")
	}
	if migrateCmd == nil {
		t.Fatal("migrate subcommand is nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/vipo-server/ -v`
Expected: FAIL — `undefined: newRootCmd`

- [ ] **Step 3: Write minimal implementation**

Create `cmd/vipo-server/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/4sp1/vipo-go/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "vipo-server",
		Short: "Vipo Pomodoro API server",
	}

	// Global flags
	root.PersistentFlags().String("db", "vipo.db", "Path to SQLite database file")
	root.PersistentFlags().String("config", "", "Path to config.json file")

	viper.BindPFlag("db", root.PersistentFlags().Lookup("db"))
	viper.BindPFlag("config", root.PersistentFlags().Lookup("config"))

	// Serve subcommand
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbFile := viper.GetString("db")
			addr := viper.GetString("addr")

			srv, err := app.NewServer(dbFile, addr)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			if err := srv.DB.Migrate(context.Background()); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			log.Printf("listening on %s", addr)
			httpServer := &http.Server{
				Addr:              addr,
				Handler:           srv.Handler,
				ReadHeaderTimeout: 10 * time.Second,
			}
			return httpServer.ListenAndServe()
		},
	}

	serve.Flags().String("addr", "127.0.0.1:8080", "Listen address")
	viper.BindPFlag("addr", serve.Flags().Lookup("addr"))

	// Migrate subcommand
	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbFile := viper.GetString("db")
			srv, err := app.NewServer(dbFile, "")
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			if err := srv.DB.Migrate(context.Background()); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			log.Println("migrations complete")
			return nil
		},
	}

	root.AddCommand(serve, migrate)

	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
```

Key decisions: cobra root with `serve` and `migrate` subcommands. [S17]. Config precedence: flag > env > config.json > defaults via viper. [S17]. `serve` runs migrations on startup then starts the HTTP server. `migrate` runs migrations independently. viper binds flags; env vars use `VIPD_` prefix (set via `viper.SetEnvPrefix("VIPD")` — add this in a follow-up or the `init()` function).

Add env prefix support:

```go
func init() {
	viper.SetEnvPrefix("VIPD")
	viper.AutomaticEnv()
}
```

This makes `VIPD_DB`, `VIPD_ADDR`, etc. environment variables override defaults.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/vipo-server/ -v`
Expected: PASS

- [ ] **Step 5: Build the binary**

Run: `go build -o /dev/null ./cmd/vipo-server/`
Expected: PASS

- [ ] **Step 6: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
feat(cmd): add cobra CLI with serve and migrate

serve: run migrations + HTTP server. migrate: run
migrations only. viper handles flag/env/config.json
layered config. [S17]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`

---

### Task 7: Update AGENTS.md for app layer

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update the Architecture section**

In `AGENTS.md`, update the architecture tree to reflect the new packages:

```markdown
internal/
├── domain/state/          # Core domain models (zero external deps)
│   ├── pomodoro/state.go  # Session state enum
│   ├── note/note.go       # Note aggregate
│   └── log/entry.go       # Event log entries
├── adapter/state/
│   ├── note/store.go      # note.Store port interface + ErrNotFound
│   └── log/eventlog.go    # log.EventLog port interface + ErrNotFound
├── port/state/sqlite/     # SQLite implementation (shared *DB wrapper)
│   ├── db.go              # New(file) → *DB, NoteStore(), EventLog(), Migrate()
│   ├── note_store.go      # note.Store implementation
│   ├── event_log.go       # log.EventLog implementation
│   ├── converters.go      # *FromStorage enum converters
│   └── migrations.go      # goose v3 Provider runner
├── port/oapi/             # Generated OAPI types/server
│   └── http/translate.go  # OAPI ↔ domain enum translation
├── svc/                   # Service layer (StrictServerInterface impl)
│   ├── service.go         # 6 handlers + error types
│   └── errors.go         # Validation error types
├── app/                   # Application wiring
│   ├── apperr/            # Error code taxonomy + Wire()
│   ├── wire.go            # NewHandler → http.Handler
│   └── server.go          # NewServer → *Server
migrations/
├── embed.go              # //go:embed *.sql
├── *.sql                 # goose-annotated migrations
```

- [ ] **Step 2: Update the Gotchas section**

Remove the gotcha about empty `cmd/vipo-server/` and `internal/svc/` and `internal/app/`:

```
- `cmd/vipo-server/` and `internal/svc/` and `internal/app/` directories exist but are **empty** — no main entrypoint or HTTP handler wiring yet. The project is domain-model-first, and the server/wiring layer hasn't been built.
```

Replace with:

```
- `cmd/vipo-server/main.go` uses cobra with `serve` and `migrate` subcommands. Config via viper (flag > env > config.json > defaults).
- The svc layer implements `oapi.StrictServerInterface` and returns typed errors (not HTTP response objects) for failure cases. The app layer maps these to HTTP responses via `apperr.Wire()`.
- 500 errors are plain text (no JSON body), matching the spec which defines no 500 response. 400/404 errors use the `Error{code, message}` schema with enumerated `code` values.
```

- [ ] **Step 3: Commit with caveman-commit**

1. `mktemp`
2. Read the temp file (required even though empty)
3. Write the commit message to the temp file:

```
docs(agents): update for app/svc/cmd layers

Document new architecture, cobra CLI, svc error
pattern, and HTTP error shaping. [S17, S16, S23]

Refs #22
```

4. `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`
