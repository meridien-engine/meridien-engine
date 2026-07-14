package hitl_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/mera/hitl"
)

type mockRepo struct {
	mu                 sync.Mutex
	expiredSuspensions []hitl.ExpiredSuspension
	fetchErr           error
	timedOutTraces     map[uuid.UUID]bool
	timeoutErr         error
}

func (m *mockRepo) GetExpiredSuspensions(ctx context.Context) ([]hitl.ExpiredSuspension, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.expiredSuspensions, nil
}

func (m *mockRepo) MarkTimedOut(ctx context.Context, traceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.timeoutErr != nil {
		return m.timeoutErr
	}
	m.timedOutTraces[traceID] = true
	return nil
}

func TestChecker_RunOnce_Success(t *testing.T) {
	traceID := uuid.New()
	repo := &mockRepo{
		expiredSuspensions: []hitl.ExpiredSuspension{
			{
				TraceID:    traceID,
				WorkflowID: "wf-1",
				ExpiresAt:  time.Now().Add(-1 * time.Hour),
			},
		},
		timedOutTraces: make(map[uuid.UUID]bool),
	}

	// We use a short interval so we can run/cancel the loop quickly.
	interval := 10 * time.Millisecond
	checker := hitl.New(repo, interval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go checker.Run(ctx)

	// Wait for a few ticks to trigger runOnce.
	time.Sleep(30 * time.Millisecond)
	cancel()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if !repo.timedOutTraces[traceID] {
		t.Errorf("expected trace %s to be marked timed out", traceID)
	}
}

func TestChecker_RunOnce_FetchError(t *testing.T) {
	repo := &mockRepo{
		fetchErr:       errors.New("db error"),
		timedOutTraces: make(map[uuid.UUID]bool),
	}

	checker := hitl.New(repo, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go checker.Run(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel()

	// Should not panic, should just log and continue.
}

func TestChecker_RunOnce_MarkError(t *testing.T) {
	traceID := uuid.New()
	repo := &mockRepo{
		expiredSuspensions: []hitl.ExpiredSuspension{
			{
				TraceID:    traceID,
				WorkflowID: "wf-1",
				ExpiresAt:  time.Now().Add(-1 * time.Hour),
			},
		},
		timeoutErr:     errors.New("db save error"),
		timedOutTraces: make(map[uuid.UUID]bool),
	}

	checker := hitl.New(repo, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go checker.Run(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel()

	// Should not mark it and should not crash.
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.timedOutTraces[traceID] {
		t.Error("expected trace not to be marked timed out due to save error")
	}
}
