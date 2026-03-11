package handlers

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

// HealthCheckProxy fetches an external health-check URL on behalf of the
// frontend (avoiding CORS issues) and returns the upstream status, latency,
// and body.
func (h *Handlers) HealthCheckProxy(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		writeError(w, http.StatusBadRequest, "url query parameter is required")
		return
	}

	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "url must be an absolute http or https URL")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	start := time.Now()
	resp, err := client.Get(target)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"reachable": false,
			"error":     err.Error(),
			"latencyMs": latencyMs,
		})
		return
	}
	defer resp.Body.Close()

	// Read up to 4KB of the response body for display.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	writeJSON(w, http.StatusOK, map[string]any{
		"reachable":  resp.StatusCode >= 200 && resp.StatusCode < 400,
		"statusCode": resp.StatusCode,
		"latencyMs":  latencyMs,
		"body":       string(body),
	})
}
