package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/domain"
)

// InteractionRepository implements domain.InteractionRepository.
type InteractionRepository struct {
	q  *db.Queries
	db *sql.DB
}

func NewInteractionRepository(database *sql.DB, q *db.Queries) *InteractionRepository {
	return &InteractionRepository{q: q, db: database}
}

// RecordTurn persists the interaction log and its full trace atomically.
// This is called after every Mera response to ensure Compass is up to date.
func (r *InteractionRepository) RecordTurn(
	ctx context.Context,
	log *domain.InteractionLog,
	trace *domain.InteractionTrace,
) error {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return err
	}

	// Serialise trace JSONB fields.
	contextsJSON, err := json.Marshal(trace.RetrievedContexts)
	if err != nil {
		return fmt.Errorf("marshal retrieved contexts: %w", err)
	}
	toolsJSON, err := json.Marshal(trace.ToolsCalled)
	if err != nil {
		return fmt.Errorf("marshal tools called: %w", err)
	}

	return ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		qtx := r.q.WithTx(tx)

		bid, _ := uuid.Parse(businessID)
		logRow, err := qtx.CreateInteractionLog(ctx, db.CreateInteractionLogParams{
			BusinessID:  bid,
			CustomerID:  log.CustomerID,
			Channel:     string(log.Channel),
			InboundMsg:  log.InboundMsg,
			OutboundMsg: log.OutboundMsg,
			TokensUsed:  log.TokensUsed,
			LatencyMs:   log.LatencyMs,
		})
		if err != nil {
			return fmt.Errorf("create interaction log: %w", err)
		}
		log.ID = logRow.ID
		log.CreatedAt = logRow.CreatedAt

		_, err = qtx.CreateInteractionTrace(ctx, db.CreateInteractionTraceParams{
			InteractionLogID:  logRow.ID,
			RetrievedContexts: contextsJSON,
			SystemPrompt:      trace.SystemPrompt,
			RawAgentThoughts:  trace.RawAgentThoughts,
			ToolsCalled:       toolsJSON,
		})
		if err != nil {
			return fmt.Errorf("create interaction trace: %w", err)
		}
		return nil
	})
}

func (r *InteractionRepository) GetWithTrace(
	ctx context.Context,
	id uuid.UUID,
) (*domain.InteractionLog, *domain.InteractionTrace, error) {
	row, err := r.q.GetInteractionWithTrace(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, domain.ErrNotFound
		}
		return nil, nil, fmt.Errorf("get interaction with trace: %w", err)
	}

	log := &domain.InteractionLog{
		ID:          row.ID,
		BusinessID:  row.BusinessID,
		CustomerID:  row.CustomerID,
		Channel:     domain.ChannelType(row.Channel),
		InboundMsg:  row.InboundMsg,
		OutboundMsg: row.OutboundMsg,
		TokensUsed:  row.TokensUsed,
		LatencyMs:   row.LatencyMs,
		CreatedAt:   row.CreatedAt,
	}

	trace := &domain.InteractionTrace{
		SystemPrompt:     row.SystemPrompt.String,
		RawAgentThoughts: row.RawAgentThoughts.String,
	}

	if row.RetrievedContexts != nil {
		_ = json.Unmarshal(row.RetrievedContexts, &trace.RetrievedContexts)
	}
	if row.ToolsCalled != nil {
		_ = json.Unmarshal(row.ToolsCalled, &trace.ToolsCalled)
	}

	return log, trace, nil
}

func (r *InteractionRepository) List(ctx context.Context, limit, offset int32) ([]domain.InteractionLog, error) {
	rows, err := r.q.ListInteractionLogs(ctx, db.ListInteractionLogsParams{
		Lim: limit,
		Off: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list interaction logs: %w", err)
	}
	out := make([]domain.InteractionLog, len(rows))
	for i, row := range rows {
		out[i] = domain.InteractionLog{
			ID:          row.ID,
			BusinessID:  row.BusinessID,
			CustomerID:  row.CustomerID,
			Channel:     domain.ChannelType(row.Channel),
			InboundMsg:  row.InboundMsg,
			OutboundMsg: row.OutboundMsg,
			TokensUsed:  row.TokensUsed,
			LatencyMs:   row.LatencyMs,
			CreatedAt:   row.CreatedAt,
		}
	}
	return out, nil
}

func (r *InteractionRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.InteractionLog, error) {
	rows, err := r.q.ListInteractionLogsByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list interactions by customer: %w", err)
	}
	out := make([]domain.InteractionLog, len(rows))
	for i, row := range rows {
		out[i] = domain.InteractionLog{
			ID:          row.ID,
			BusinessID:  row.BusinessID,
			CustomerID:  row.CustomerID,
			Channel:     domain.ChannelType(row.Channel),
			InboundMsg:  row.InboundMsg,
			OutboundMsg: row.OutboundMsg,
			TokensUsed:  row.TokensUsed,
			LatencyMs:   row.LatencyMs,
			CreatedAt:   row.CreatedAt,
		}
	}
	return out, nil
}
