// Package hitl implements Human-in-the-Loop suspension tracking and
// expiry enforcement for the Mera workflow engine.
package hitl

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ExpiredSuspension is the minimal projection the checker needs.
type ExpiredSuspension struct {
	TraceID    uuid.UUID
	WorkflowID string
	ExpiresAt  time.Time
}

// HITLRepository is the data access interface the checker depends on.
// The concrete implementation is in repository.InteractionRepository.
type HITLRepository interface {
	GetExpiredSuspensions(ctx context.Context) ([]ExpiredSuspension, error)
	MarkTimedOut(ctx context.Context, traceID uuid.UUID) error
}

// Checker polls the database for expired HITL suspensions and
// auto-rejects them, keeping merchant dashboards accurate.
type Checker struct {
	repo     HITLRepository
	interval time.Duration
}

// New creates a Checker with the given poll interval.
// Use 5*time.Minute as the default in production.
func New(repo HITLRepository, interval time.Duration) *Checker {
	return &Checker{repo: repo, interval: interval}
}

// Run starts the background polling loop. It blocks until ctx is cancelled.
// Call this in a goroutine from main.go.
func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	slog.Info("hitl checker started", "interval", c.interval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("hitl checker stopped")
			return
		case <-ticker.C:
			c.runOnce(ctx)
		}
	}
}

func (c *Checker) runOnce(ctx context.Context) {
	expired, err := c.repo.GetExpiredSuspensions(ctx)
	if err != nil {
		slog.Error("hitl checker: failed to fetch expired suspensions", "error", err)
		return
	}

	for _, s := range expired {
		if err := c.repo.MarkTimedOut(ctx, s.TraceID); err != nil {
			slog.Error("hitl checker: failed to mark timed_out",
				"trace_id", s.TraceID,
				"workflow_id", s.WorkflowID,
				"error", err,
			)
			continue
		}
		slog.Warn("hitl: workflow timed out, auto-rejected",
			"trace_id", s.TraceID,
			"workflow_id", s.WorkflowID,
			"expired_at", s.ExpiresAt,
		)
	}
}
