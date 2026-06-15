// Package health provides liveness and readiness HTTP handlers.
//
// Endpoints:
//   GET /healthz   — liveness probe  (always 200 if the process is alive)
//   GET /readyz    — readiness probe (200 only when DB is reachable)
//
// These are consumed by Docker healthchecks, Kubernetes probes, and
// the Prometheus blackbox exporter.
package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// Checker holds dependencies needed to perform deep health checks.
type Checker struct {
	db      *sql.DB
	version string // injected at build time via -ldflags
}

func New(db *sql.DB, version string) *Checker {
	return &Checker{db: db, version: version}
}

// response is the JSON body returned by both health endpoints.
type response struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// Liveness reports that the process is alive.
// Never touches the database — always returns 200 while the process runs.
func (c *Checker) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, response{
		Status:  "ok",
		Version: c.version,
	})
}

// Readiness checks that all critical dependencies are reachable.
// Returns 503 if any check fails so load balancers can stop routing traffic.
func (c *Checker) Readiness(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{}
	overallOK := true

	// ── Database ping ──────────────────────────────────────────────────────────
	if c.db == nil {
		checks["postgres"] = "unhealthy: database connection not configured"
		overallOK = false
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := c.db.PingContext(ctx); err != nil {
			checks["postgres"] = "unhealthy: " + err.Error()
			overallOK = false
		} else {
			checks["postgres"] = "ok"
		}
	}

	status := "ok"
	code := http.StatusOK
	if !overallOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, response{
		Status:  status,
		Version: c.version,
		Checks:  checks,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
