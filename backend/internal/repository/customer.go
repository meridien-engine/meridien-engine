package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/domain"
)

// CustomerRepository implements domain.CustomerRepository using sqlc + Postgres.
type CustomerRepository struct {
	q  *db.Queries
	db *sql.DB
}

func NewCustomerRepository(database *sql.DB, q *db.Queries) *CustomerRepository {
	return &CustomerRepository{q: q, db: database}
}

// GetOrCreateByChannel resolves a customer profile from an external channel
// identifier (e.g. WhatsApp phone number). If no profile exists yet, it
// creates one atomically and returns isNew=true.
func (r *CustomerRepository) GetOrCreateByChannel(
	ctx context.Context,
	channel domain.ChannelType,
	externalID string,
) (*domain.CustomerProfile, bool, error) {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return nil, false, err
	}

	// Fast path: try to find existing profile.
	existing, err := r.q.GetCustomerByChannel(ctx, db.GetCustomerByChannelParams{
		ChannelType:       string(channel),
		ChannelExternalID: externalID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("get customer by channel: %w", err)
	}
	if err == nil {
		return mapCustomerProfile(existing), false, nil
	}

	// Slow path: create new profile + channel mapping inside a transaction.
	bid, _ := uuid.Parse(businessID)
	var profile *domain.CustomerProfile

	err = ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		qtx := r.q.WithTx(tx)

		row, err := qtx.CreateCustomerProfile(ctx, db.CreateCustomerProfileParams{
			BusinessID:   bid,
			UnifiedName:  sql.NullString{},
			CustomerTier: "standard",
		})
		if err != nil {
			return fmt.Errorf("create customer profile: %w", err)
		}

		_, err = qtx.UpsertCustomerChannel(ctx, db.UpsertCustomerChannelParams{
			CustomerProfileID: row.ID,
			ChannelType:       string(channel),
			ChannelExternalID: externalID,
		})
		if err != nil {
			return fmt.Errorf("upsert customer channel: %w", err)
		}

		profile = mapCustomerProfile(row)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return profile, true, nil
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	row, err := r.q.GetCustomerProfileByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get customer profile: %w", err)
	}
	return mapCustomerProfile(row), nil
}

func (r *CustomerRepository) UpdateSemanticSummary(ctx context.Context, id uuid.UUID, summary string) error {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return err
	}
	return ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		_, err := r.q.WithTx(tx).UpdateSemanticSummary(ctx, db.UpdateSemanticSummaryParams{
			ID:              id,
			SemanticSummary: sql.NullString{String: summary, Valid: true},
		})
		return err
	})
}

func (r *CustomerRepository) UpdateTier(ctx context.Context, id uuid.UUID, tier domain.CustomerTier) error {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return err
	}
	return ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		_, err := r.q.WithTx(tx).UpdateCustomerTier(ctx, db.UpdateCustomerTierParams{
			ID:           id,
			CustomerTier: string(tier),
		})
		return err
	})
}

// ─── mapper ───────────────────────────────────────────────────────────────────

func mapCustomerProfile(row db.CustomerProfile) *domain.CustomerProfile {
	return &domain.CustomerProfile{
		ID:              row.ID,
		BusinessID:      row.BusinessID,
		UnifiedName:     row.UnifiedName.String,
		CustomerTier:    domain.CustomerTier(row.CustomerTier),
		SemanticSummary: row.SemanticSummary.String,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
