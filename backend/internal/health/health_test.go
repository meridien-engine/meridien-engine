package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meridien-engine/meridien-engine/internal/health"
)

// TestLiveness_AlwaysReturns200 verifies that the liveness probe returns 200
// unconditionally — it should never check the database.
func TestLiveness_AlwaysReturns200(t *testing.T) {
	// Pass nil db — liveness must NOT touch the database.
	checker := health.New(nil, "v0.1.0-test")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	checker.Liveness(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
	if body["version"] != "v0.1.0-test" {
		t.Errorf("expected version v0.1.0-test, got %v", body["version"])
	}
}

// TestReadiness_NilDB_Returns503 verifies that when the database is nil (not
// configured or unreachable), readiness returns 503.
func TestReadiness_NilDB_Returns503(t *testing.T) {
	checker := health.New(nil, "dev")

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	checker.Readiness(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON body: %v", err)
	}

	if body["status"] != "degraded" {
		t.Errorf("expected status degraded, got %v", body["status"])
	}
}
