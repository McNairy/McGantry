// Package metrics provides a lightweight Prometheus-compatible /metrics endpoint.
// It uses the standard text exposition format without the prometheus/client_golang
// dependency, keeping the binary footprint small.
package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---- Counter ----------------------------------------------------------------

// Counter is a monotonically increasing int64 counter with labels.
type Counter struct {
	mu   sync.RWMutex
	vals map[string]int64
}

func NewCounter() *Counter { return &Counter{vals: make(map[string]int64)} }

// Inc increments the counter for the given label set by 1.
func (c *Counter) Inc(labels map[string]string) {
	k := labelsKey(labels)
	c.mu.Lock()
	c.vals[k]++
	c.mu.Unlock()
}

// ---- Gauge ------------------------------------------------------------------

// Gauge is a settable int64 value with labels.
type Gauge struct {
	mu   sync.RWMutex
	vals map[string]int64
}

func NewGauge() *Gauge { return &Gauge{vals: make(map[string]int64)} }

// Set sets the gauge to v for the given label set.
func (g *Gauge) Set(labels map[string]string, v int64) {
	k := labelsKey(labels)
	g.mu.Lock()
	g.vals[k] = v
	g.mu.Unlock()
}

// ---- Histogram (simplified: sum + count + buckets) -------------------------

var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}

type histogramSeries struct {
	sum     float64
	count   int64
	buckets []int64 // one per defaultBuckets entry
}

// Histogram tracks request durations.
type Histogram struct {
	mu   sync.Mutex
	vals map[string]*histogramSeries
}

func NewHistogram() *Histogram { return &Histogram{vals: make(map[string]*histogramSeries)} }

// Observe records a duration observation for the given label set.
func (h *Histogram) Observe(labels map[string]string, d time.Duration) {
	secs := d.Seconds()
	k := labelsKey(labels)
	h.mu.Lock()
	s, ok := h.vals[k]
	if !ok {
		s = &histogramSeries{buckets: make([]int64, len(defaultBuckets))}
		h.vals[k] = s
	}
	s.sum += secs
	s.count++
	for i, bound := range defaultBuckets {
		if secs <= bound {
			s.buckets[i]++
		}
	}
	h.mu.Unlock()
}

// ---- Registry ---------------------------------------------------------------

// Registry holds all named metrics and writes the prometheus text format.
type Registry struct {
	mu         sync.RWMutex
	counters   map[string]*counterEntry
	gauges     map[string]*gaugeEntry
	histograms map[string]*histogramEntry
}

type counterEntry struct {
	help    string
	counter *Counter
}

type gaugeEntry struct {
	help  string
	gauge *Gauge
}

type histogramEntry struct {
	help      string
	histogram *Histogram
}

func NewRegistry() *Registry {
	return &Registry{
		counters:   make(map[string]*counterEntry),
		gauges:     make(map[string]*gaugeEntry),
		histograms: make(map[string]*histogramEntry),
	}
}

func (r *Registry) RegisterCounter(name, help string) *Counter {
	c := NewCounter()
	r.mu.Lock()
	r.counters[name] = &counterEntry{help: help, counter: c}
	r.mu.Unlock()
	return c
}

func (r *Registry) RegisterGauge(name, help string) *Gauge {
	g := NewGauge()
	r.mu.Lock()
	r.gauges[name] = &gaugeEntry{help: help, gauge: g}
	r.mu.Unlock()
	return g
}

func (r *Registry) RegisterHistogram(name, help string) *Histogram {
	h := NewHistogram()
	r.mu.Lock()
	r.histograms[name] = &histogramEntry{help: help, histogram: h}
	r.mu.Unlock()
	return h
}

// Handler returns an http.Handler that writes the prometheus text format.
// onCollect is called before writing to allow callers to refresh gauges (e.g. DB counts).
func (r *Registry) Handler(onCollect func()) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if onCollect != nil {
			onCollect()
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		var sb strings.Builder

		r.mu.RLock()
		// Write counters.
		for name, e := range r.counters {
			fmt.Fprintf(&sb, "# HELP %s %s\n# TYPE %s counter\n", name, e.help, name)
			e.counter.mu.RLock()
			for k, v := range e.counter.vals {
				fmt.Fprintf(&sb, "%s{%s} %d\n", name, k, v)
			}
			e.counter.mu.RUnlock()
		}
		// Write gauges.
		for name, e := range r.gauges {
			fmt.Fprintf(&sb, "# HELP %s %s\n# TYPE %s gauge\n", name, e.help, name)
			e.gauge.mu.RLock()
			for k, v := range e.gauge.vals {
				fmt.Fprintf(&sb, "%s{%s} %d\n", name, k, v)
			}
			e.gauge.mu.RUnlock()
		}
		// Write histograms.
		for name, e := range r.histograms {
			fmt.Fprintf(&sb, "# HELP %s %s\n# TYPE %s histogram\n", name, e.help, name)
			e.histogram.mu.Lock()
			for k, s := range e.histogram.vals {
				for i, bound := range defaultBuckets {
					le := fmt.Sprintf("%g", bound)
					bucketLabels := appendLabel(k, "le", le)
					fmt.Fprintf(&sb, "%s_bucket{%s} %d\n", name, bucketLabels, s.buckets[i])
				}
				infLabels := appendLabel(k, "le", "+Inf")
				fmt.Fprintf(&sb, "%s_bucket{%s} %d\n", name, infLabels, s.count)
				fmt.Fprintf(&sb, "%s_sum{%s} %g\n", name, k, s.sum)
				fmt.Fprintf(&sb, "%s_count{%s} %d\n", name, k, s.count)
			}
			e.histogram.mu.Unlock()
		}
		r.mu.RUnlock()

		fmt.Fprint(w, sb.String())
	})
}

// ---- Default registry -------------------------------------------------------

var (
	defaultRegistry = NewRegistry()

	// APIRequestsTotal counts HTTP requests by method, path, and status.
	APIRequestsTotal = defaultRegistry.RegisterCounter(
		"gantry_api_requests_total",
		"Total number of API requests, partitioned by method, path, and status.",
	)

	// APIRequestDuration tracks API request latency.
	APIRequestDuration = defaultRegistry.RegisterHistogram(
		"gantry_api_request_duration_seconds",
		"API request latency in seconds, partitioned by method and path.",
	)

	// EntitiesTotal tracks entity counts by kind.
	EntitiesTotal = defaultRegistry.RegisterGauge(
		"gantry_entities_total",
		"Total number of entities in the catalog, partitioned by kind.",
	)

	// WebSocketConnections tracks active WebSocket connections.
	WebSocketConnections int64 // updated atomically by ws hub
	wsConnectionsGauge   = defaultRegistry.RegisterGauge(
		"gantry_websocket_connections",
		"Number of active WebSocket connections.",
	)
)

// IncWebSocketConnections increments the WebSocket connection gauge.
func IncWebSocketConnections() {
	atomic.AddInt64(&WebSocketConnections, 1)
	wsConnectionsGauge.Set(nil, atomic.LoadInt64(&WebSocketConnections))
}

// DecWebSocketConnections decrements the WebSocket connection gauge.
func DecWebSocketConnections() {
	atomic.AddInt64(&WebSocketConnections, -1)
	wsConnectionsGauge.Set(nil, atomic.LoadInt64(&WebSocketConnections))
}

// Handler returns the default registry's HTTP handler.
func Handler(onCollect func()) http.Handler {
	return defaultRegistry.Handler(onCollect)
}

// ---- helpers ----------------------------------------------------------------

// labelsKey converts a label map to a stable prometheus label string.
func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(`%s=%q`, k, labels[k]))
	}
	return strings.Join(parts, ",")
}

// appendLabel appends a single label to an existing label string.
func appendLabel(existing, k, v string) string {
	part := fmt.Sprintf(`%s=%q`, k, v)
	if existing == "" {
		return part
	}
	return existing + "," + part
}
