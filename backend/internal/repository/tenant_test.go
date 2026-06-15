package repository_test

import (
	"context"
	"testing"

	"github.com/meridien-engine/meridien-engine/internal/repository"
)

func TestWithBusinessID_RoundTrips(t *testing.T) {
	ctx := context.Background()
	bid := "550e8400-e29b-41d4-a716-446655440000"

	ctx = repository.WithBusinessID(ctx, bid)

	got, err := repository.BusinessIDFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != bid {
		t.Errorf("expected %q, got %q", bid, got)
	}
}

func TestBusinessIDFromContext_EmptyContext_ReturnsError(t *testing.T) {
	ctx := context.Background()

	_, err := repository.BusinessIDFromContext(ctx)
	if err == nil {
		t.Fatal("expected error for empty context")
	}
}

func TestWithBusinessID_EmptyString_ReturnsError(t *testing.T) {
	ctx := repository.WithBusinessID(context.Background(), "")

	_, err := repository.BusinessIDFromContext(ctx)
	if err == nil {
		t.Fatal("expected error for empty business ID")
	}
}
