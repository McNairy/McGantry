// Package middleware provides HTTP middleware for the Gantry API server.
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go2engle/gantry/internal/metrics"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLogger logs each request as a structured JSON line to stderr.
// Fields: time, level, msg, method, path, status, duration_ms, request_id.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		dur := time.Since(start)
		status := wrapped.statusCode

		// Record metrics.
		metrics.APIRequestsTotal.Inc(map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": fmt.Sprintf("%d", status),
		})
		metrics.APIRequestDuration.Observe(map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
		}, dur)

		entry := map[string]any{
			"time":        time.Now().UTC().Format(time.RFC3339),
			"level":       "info",
			"msg":         "request",
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      status,
			"duration_ms": float64(dur.Microseconds()) / 1000.0,
			"request_id":  chimiddleware.GetReqID(r.Context()),
		}

		if b, err := json.Marshal(entry); err == nil {
			fmt.Fprintln(os.Stderr, string(b))
		}
	})
}
