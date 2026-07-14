package metrics_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meridien-engine/meridien-engine/internal/metrics"
)

func TestMetricsMiddleware_IncrementsCounter(t *testing.T) {
	// Setup dummy handler to be wrapped by the metrics middleware
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	mMiddleware := metrics.Middleware(dummyHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	mMiddleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	// Read the metrics output from the prometheus handler
	metricsHandler := metrics.Handler()
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRR := httptest.NewRecorder()
	metricsHandler.ServeHTTP(metricsRR, metricsReq)

	bodyBytes, err := io.ReadAll(metricsRR.Body)
	if err != nil {
		t.Fatalf("failed to read metrics body: %v", err)
	}

	bodyStr := string(bodyBytes)

	// Verify that our HTTP request was registered
	// The label values will match: method="POST", path="/api/v1/test", status="Created" (since 201 is StatusCreated)
	expectedMetricLine := `meridien_http_requests_total{method="POST",path="/api/v1/test",status="Created"}`
	if !strings.Contains(bodyStr, expectedMetricLine) {
		t.Errorf("expected metrics to contain line %q, but it was not found. Output:\n%s", expectedMetricLine, bodyStr)
	}
}
