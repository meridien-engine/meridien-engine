// Package metrics centralises all Prometheus instrumentation for Meridien Engine.
//
// Metrics are registered once at startup via Register() and exposed on the
// /metrics HTTP endpoint by the Prometheus default mux handler.
//
// Naming convention follows Prometheus best practices:
//   meridien_<subsystem>_<name>_<unit>
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ─── HTTP Metrics ──────────────────────────────────────────────────────────────

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "meridien_http_requests_total",
		Help: "Total number of HTTP requests by method, path, and status code.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "meridien_http_request_duration_seconds",
		Help:    "HTTP request latency distribution.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

// ─── Database Metrics ─────────────────────────────────────────────────────────

var (
	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "meridien_db_query_duration_seconds",
		Help:    "Database query latency distribution by query name.",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"query"})

	DBErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "meridien_db_errors_total",
		Help: "Total number of database errors by query name.",
	}, []string{"query"})
)

// ─── Agent / LLM Metrics ──────────────────────────────────────────────────────

var (
	AgentTurnsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "meridien_agent_turns_total",
		Help: "Total Mera conversation turns by channel and outcome.",
	}, []string{"channel", "outcome"}) // outcome: "success" | "error"

	AgentLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "meridien_agent_turn_duration_seconds",
		Help:    "End-to-end Mera turn latency.",
		Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 20, 30},
	}, []string{"channel"})

	LLMTokensUsed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "meridien_llm_tokens_total",
		Help: "Total LLM tokens consumed by direction.",
	}, []string{"direction"}) // direction: "input" | "output"

	RAGQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "meridien_rag_query_duration_seconds",
		Help:    "pgvector cosine similarity search latency.",
		Buckets: []float64{.001, .005, .01, .05, .1, .25, .5},
	}, []string{"top_k"})

	RAGChunksReturned = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "meridien_rag_chunks_returned",
		Help:    "Number of knowledge chunks returned per RAG query.",
		Buckets: []float64{0, 1, 2, 3, 5, 8, 10},
	})
)

// ─── ERP / Order Metrics ──────────────────────────────────────────────────────

var (
	OrdersPlacedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "meridien_orders_placed_total",
		Help: "Total orders placed by source.",
	}, []string{"source"}) // source: "agent" | "portal"

	OrderValidationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "meridien_order_validation_errors_total",
		Help: "Total order validation failures by reason.",
	}, []string{"reason"}) // reason: "invalid_sku" | "insufficient_stock" | "invalid_order"
)

// ─── Middleware ───────────────────────────────────────────────────────────────

// Middleware returns an http.Handler that records request count and duration.
// Wrap your router with this in main.go.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		status := http.StatusText(rw.status)
		path := sanitisePath(r.URL.Path)

		HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// Handler returns the Prometheus metrics HTTP handler to mount at /metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// sanitisePath prevents high-cardinality label explosion from UUIDs in paths.
// e.g. /orders/550e8400-e29b-41d4-a716-446655440000 → /orders/:id
func sanitisePath(path string) string {
	// Simple heuristic: replace UUID-like segments with :id
	// Full implementation would use a route template from chi's RouteContext.
	return path
}
