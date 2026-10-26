package app

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestMiddlewareBodiesAtDebugOnly(t *testing.T) {
	const reqPayload = `{"note":"hello","pomodoro_state":"work"}`
	const respPayload = `{"id":1,"note":"hello"}`

	t.Run("debug", func(t *testing.T) {
		buf, restore := captureLogging(t)
		defer restore()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(respPayload))
		})
		req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(reqPayload))
		rec := httptest.NewRecorder()
		LoggingMiddleware(next).ServeHTTP(rec, req)

		out := buf.String()
		if !strings.Contains(out, `msg="http request body"`) || !strings.Contains(out, strconv.Quote(reqPayload)) {
			t.Errorf("debug: request body not captured, got:\n%s", out)
		}
		if !strings.Contains(out, `msg="http response body"`) || !strings.Contains(out, strconv.Quote(respPayload)) {
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
		if strings.Contains(out, strconv.Quote(reqPayload)) || strings.Contains(out, strconv.Quote(respPayload)) {
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
	if !strings.Contains(out, `status=400`) || !strings.Contains(out, `\"code\":\"invalid_request\"`) {
		t.Errorf("debug: error response body must be captured, got:\n%s", out)
	}
}
