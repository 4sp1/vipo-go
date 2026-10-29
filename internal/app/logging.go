package app

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
)

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

		slog.Warn("http access",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		slog.Debug("http request body",
			"method", r.Method, "path", r.URL.Path, "body", reqBuf.String())
		slog.Debug("http response body",
			"method", r.Method, "path", r.URL.Path, "body", rec.resp.String())
	})
}

func accessRecorderFrom(w http.ResponseWriter) *accessRecorder {
	if rec, ok := w.(*accessRecorder); ok {
		return rec
	}
	return &accessRecorder{ResponseWriter: w}
}
