package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/meridien-engine/meridien-engine/internal/mera/agent"
	"github.com/meridien-engine/meridien-engine/internal/repository"
)

// noopFn is a NodeFn that succeeds and echoes its input as output.
func noopFn(_ context.Context, input map[string]any) (map[string]any, error) {
	return input, nil
}

// failFn is a NodeFn that always returns an error.
func failFn(_ context.Context, _ map[string]any) (map[string]any, error) {
	return nil, errors.New("node failure")
}

func TestWithTrace_Success(t *testing.T) {
	// WithTrace must pass the result through transparently on success.
	ctx := repository.WithBusinessID(context.Background(), "biz-123")
	input := map[string]any{"key": "value"}

	wrapped := agent.WithTrace(agent.SpanRAGRetrieval, noopFn)
	out, err := wrapped(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out["key"] != "value" {
		t.Errorf("expected output to echo input, got %v", out)
	}
}

func TestWithTrace_ErrorPropagated(t *testing.T) {
	// WithTrace must propagate errors from the inner fn without swallowing them.
	ctx := repository.WithBusinessID(context.Background(), "biz-456")

	wrapped := agent.WithTrace(agent.SpanERPCheckout, failFn)
	_, err := wrapped(ctx, nil)

	if err == nil {
		t.Fatal("expected error to be propagated, got nil")
	}
	if err.Error() != "node failure" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestWithTrace_MissingTenant(t *testing.T) {
	// WithTrace must not panic when business_id is absent in context.
	// It should still execute the fn normally — the attribute is best-effort.
	ctx := context.Background() // no business_id injected
	wrapped := agent.WithTrace(agent.SpanLLMRoute, noopFn)
	_, err := wrapped(ctx, map[string]any{})

	if err != nil {
		t.Fatalf("expected no error even without tenant ctx, got %v", err)
	}
}
