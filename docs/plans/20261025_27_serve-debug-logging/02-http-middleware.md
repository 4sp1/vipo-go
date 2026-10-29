# 02 — HTTP middleware with debug-only body capture (issue 27) Sub-plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `complete-plan-execution` to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** One logging middleware on `srv.Handler` in `internal/app/wire.go` that logs access metadata (`method`, `path`, `status`, `duration`) for every request including unmatched routes, plus request/response bodies at debug level only.

**Architecture:** New file `internal/app/logging.go` (middleware + types) and `internal/app/logging_test.go`. The middleware calls package-level `slog` (no signature churn, per issue decision). Body capture uses the tee pattern: request side via `io.TeeReader` into a shared `bytes.Buffer` (the buffer fills only when the handler reads the body), response side via a hand-rolled `ResponseWriter` wrapper whose `Write` mirrors into a second buffer (there is no `io.Tee` for `ResponseWriter`). Two levels matter for the capture logic: the access line is emitted at `Warn` (visible at the warn default and above, satisfying the anti-leak split: only access metadata is visible above debug), the body lines at `Debug`.

Anti-leak decision consequence, recorded here to avoid relitigating: access metadata is logged at `Warn`, not `Info`, because a handler configured at `warn` (the default) would otherwise show nothing. Status is a field of the access line, not the log level.

**Tech Stack:** Go stdlib (`net/http`, `io`, `bytes`, `log/slog`, `time`), `httptest` for tests.

---

### Task 1: Access-metadata middleware (status recording)

**Files:**
- Create: `internal/app/logging.go`
- Test: `internal/app/logging_test.go` (create)
- Modify: `internal/app/wire.go` (wiring, 1 line)

- [x] **Step 1: Write the failing test**

Create `internal/app/logging_test.go`:

```go
package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func captureLogging(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	handler := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// omit time/source so output is fully deterministic
			if a.Key == slog.TimeKey || a.Key == slog.SourceKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	slog.SetDefault(handler)
	return &buf, func() { slog.SetDefault(prev) }
}

func TestMiddlewareLogsAccessMetadata(t *testing.T) {
	buf, restore := captureLogging(t)
	defer restore()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mw := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodPost, "/notes", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	out := buf.String()
	for _, want := range []string{
		`msg="http access"`,
		`method=POST`,
		`path=/notes`,
		`status=201`,
		`duration_ms=`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("access log missing %q, got:\n%s", want, out)
		}
	}
}

func TestMiddlewareLogsUnmatchedRoutesAtWarnLevel(t *testing.T) {
	buf, restore := captureLogging(t)
	defer restore()

	next := http.HandlerFunc(http.NotFound)
	mw := LoggingMiddleware(next)

	req := httptest.NewRequest(http.MethodDelete, "/definitely/unmatched", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, `status=404`) {
		t.Errorf("access log missing status=404, got:\n%s", out)
	}
	if !strings.Contains(out, `level=WARN`) {
		t.Errorf("access log must be at WARN level (visible at warn default), got:\n%s", out)
	}
}
```

Run: `go test ./internal/app/ -run 'TestMiddleware' -v`
Expected: FAIL — `LoggingMiddleware` (or `logging.go`) does not exist / undefined.

Also wire the middleware here so the whole package still compiles (the test file alone does not need it, but wiring is one line and if not done, Task 2 must remember it): in `internal/app/wire.go`, change the return line of `NewHandler` from

```go
	return oapi.Handler(strictHandler)
```

to

```go
	return LoggingMiddleware(oapi.Handler(strictHandler))
```

Run: `go build ./...`
Expected: FAIL with `undefined: LoggingMiddleware` (this is the red we want before writing code).

- [x] **Step 2: Write minimal implementation**

Create `internal/app/logging.go`:

```go
package app

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// accessRecorder records the response status and mirrors every write
// so bodies can be logged at debug level.
type accessRecorder struct {
	http.ResponseWriter
	status int
	resp   bytes.Buffer
}

func (a *accessRecorder) WriteHeader(code int) {
	if a.status == 0 {
		a.status = code
	}
	a.ResponseWriter.WriteHeader(code)
}

func (a *accessRecorder) Write(b []byte) (int, error) {
	if a.status == 0 {
		a.status = http.StatusOK
	}
	a.resp.Write(b)
	return a.ResponseWriter.Write(b)
}

func (a *accessRecorder) Flush() {
	if f, ok := a.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// LoggingMiddleware logs access metadata (method/path/status/duration) for
// every request, plus request/response bodies at debug level only.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var reqBuf bytes.Buffer
		if r.Body != nil {
			r.Body = io.NopCloser(io.TeeReader(r.Body, &reqBuf))
		}
		rec := accessRecorderFrom(w)

		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}

		access := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		slog.Warn("http access", access...)

		slog.Debug("http request body",
			"method", r.Method, "path", r.URL.Path, "body", reqBuf.String())
		slog.Debug("http response body",
			"method", r.Method, "path", r.URL.Path, "body", rec.resp.String())
	})
}

func accessRecorderFrom(w http.ResponseWriter) *accessRecorder {
	// If a recorder was somehow already installed (double wrap), reuse it to
	// keep status/body capture single-source.
	if rec, ok := w.(*accessRecorder); ok {
		return rec
	}
	return &accessRecorder{ResponseWriter: w}
}
```

In `internal/app/wire.go` change the return of `NewHandler` to `return LoggingMiddleware(oapi.Handler(strictHandler))` (if not already done in Step 1).

- [x] **Step 3: Run test to verify it passes**

Run: `go test ./internal/app/ -run 'TestMiddleware' -v && go build ./...`
Expected: PASS.

- [x] **Step 4: Commit with caveman-commit**

Use `caveman-commit` for the message content, then the commit sequence from `core-commands` (mktemp → Read temp file → Write message → `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`).

Message sketch: `feat(app): http access middleware` (subject ≤50 chars).

---

### Task 2: Body capture is debug-only (anti-leak proof)

**Files:**
- Modify: `internal/app/logging_test.go`
- (behavior under test in `internal/app/logging.go`, already implemented)

- [x] **Step 1: Write the failing tests (they enforce bodies at debug and silence above)**

Append to `internal/app/logging_test.go`:

```go
func TestMiddlewareBodiesAtDebugOnly(t *testing.T) {
	const reqPayload = `{"note":"hello","pomodoro_state":"work"}`
	const respPayload = `{"id":1,"note":"hello"}`

	t.Run("debug", func(t *testing.T) {
		buf, restore := captureLogging(t)
		defer restore()

		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(respPayload))
		})
		req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(reqPayload))
		rec := httptest.NewRecorder()
		LoggingMiddleware(next).ServeHTTP(rec, req)

		out := buf.String()
		if !strings.Contains(out, `msg="http request body"`) || !strings.Contains(out, reqPayload) {
			t.Errorf("debug: request body not captured, got:\n%s", out)
		}
		if !strings.Contains(out, `msg="http response body"`) || !strings.Contains(out, respPayload) {
			t.Errorf("debug: response body not captured, got:\n%s", out)
		}
	})

	t.Run("warn default hides bodies but keeps metadata", func(t *testing.T) {
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
		defer slog.SetDefault(prev)

		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(respPayload))
		})
		req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(reqPayload))
		rec := httptest.NewRecorder()
		LoggingMiddleware(next).ServeHTTP(rec, req)

		out := buf.String()
		if !strings.Contains(out, `msg="http access"`) {
			t.Errorf("warn: access line missing, got:\n%s", out)
		}
		if strings.Contains(out, reqPayload) || strings.Contains(out, respPayload) {
			t.Errorf("warn: body content leaked, got:\n%s", out)
		}
	})
}

func TestMiddlewareErrorBodyAtDebugHierarchy(t *testing.T) {
	buf, restore := captureLogging(t)
	defer restore()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"code":"invalid_request","message":"boom"}`, http.StatusBadRequest)
	})
	req := httptest.NewRequest(http.MethodGet, "/notes/0", nil)
	rec := httptest.NewRecorder()
	LoggingMiddleware(next).ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, `status=400`) || !strings.Contains(out, `"code":"invalid_request"`) {
		t.Errorf("debug: error response body must be captured, got:\n%s", out)
	}
}
```

Note: the body tests assert via `slog.TextHandler` output, which quotes values; `strings.Contains` of the single-line JSON payload is still exact and sufficient here.

- [x] **Step 2: Run tests**

Run: `go test ./internal/app/ -run 'TestMiddleware' -v`
Expected: PASS (implementation already supports both).

- [x] **Step 3: Full suite**

Run: `go test ./... -count=1 && go vet ./...`
Expected: PASS. Address any test in `internal/app/e2e_test.go` that now logs into the test console (existing tests may now emit `http access` lines per request; that is harmless on stdout during tests, silence them by leaving default level alone).

- [x] **Step 4: Commit with caveman-commit**

Use `caveman-commit` then the `core-commands` commit sequence (mktemp → Read temp file → Write message → `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`).

Message sketch: `test(app): bodies debug-only assertions` (subject ≤50 chars).

---

### Task 3: Anti-leak acceptance check at e2e level

**Files:**
- Test: `internal/app/e2e_test.go` (modify; append one test)

- [x] **Step 1: Write the failing test**

Append to `internal/app/e2e_test.go`:

```go
func TestDebugModeLogsBodiesE2E(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	srv := setupTestServer(t)

	body := `{"note":"e2e-knock","pomodoro_state":"work"}`
	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	out := buf.String()
	if !strings.Contains(out, `"msg":"http access"`) || !strings.Contains(out, `"msg":"http request body"`) {
		t.Errorf("debug mode missing access/body lines, got:\n%s", out)
	}
}
```

If `e2e_test.go` does not already import `bytes`, `log/slog`, `net/http/httptest`, `strings`, add these imports.

- [x] **Step 2: Run test to verify it passes**

Run: `go test ./internal/app/ -run 'TestDebugModeLogsBodiesE2E' -v`
Expected: PASS (the middleware, wired into `srv.Handler` by `wire.go`, already produces these lines; this test keeps AC 4 + AC 6 pinned at the real handler boundary).

- [x] **Step 3: Full suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [x] **Step 4: Commit with caveman-commit**

Use `caveman-commit` then the `core-commands` commit sequence (mktemp → Read temp file → Write message → `jj describe --stdin < "/tmp/tmp.XXXXXX" && rm "/tmp/tmp.XXXXXX" && jj new`).

Message sketch: `test(app): e2e debug body capture` (subject ≤50 chars).
