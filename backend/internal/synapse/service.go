// Package synapse implements the customer identity and interaction
// recording service — the Synapse layer of Meridien Engine.
package synapse

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
)

// Service handles UCM resolution and interaction tracing.
type Service struct {
	customers    domain.CustomerRepository
	interactions domain.InteractionRepository
}

func NewService(customers domain.CustomerRepository, interactions domain.InteractionRepository) *Service {
	return &Service{customers: customers, interactions: interactions}
}

// GetOrCreateCustomer resolves a UCM record from an inbound channel identifier.
// If no matching profile exists, a new one is created and returned.
// This is the first call Mera makes on every inbound message.
func (s *Service) GetOrCreateCustomer(
	ctx context.Context,
	channel domain.ChannelType,
	externalID string,
) (*domain.CustomerProfile, bool, error) {
	profile, isNew, err := s.customers.GetOrCreateByChannel(ctx, channel, externalID)
	if err != nil {
		return nil, false, fmt.Errorf("resolve customer profile: %w", err)
	}
	return profile, isNew, nil
}

// RecordTurn persists the full interaction log and its observability trace.
// Called by Mera after producing its response, synchronously before replying
// to the customer so that Compass is always up-to-date.
func (s *Service) RecordTurn(
	ctx context.Context,
	log *domain.InteractionLog,
	trace *domain.InteractionTrace,
) error {
	// Ensure IDs are set if caller hasn't done so.
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if trace.ID == uuid.Nil {
		trace.ID = uuid.New()
	}
	trace.InteractionLogID = log.ID

	if err := s.interactions.RecordTurn(ctx, log, trace); err != nil {
		return fmt.Errorf("record interaction turn: %w", err)
	}
	return nil
}

// UpdateSemanticSummary overwrites the rolling UCM summary for a customer.
// Called by the background Synapse summarizer after processing recent logs.
func (s *Service) UpdateSemanticSummary(ctx context.Context, customerID uuid.UUID, summary string) error {
	if err := s.customers.UpdateSemanticSummary(ctx, customerID, summary); err != nil {
		return fmt.Errorf("update semantic summary: %w", err)
	}
	return nil
}

// GetInteractionTrace fetches a full interaction turn with its trace for Compass.
func (s *Service) GetInteractionTrace(
	ctx context.Context,
	logID uuid.UUID,
) (*domain.InteractionLog, *domain.InteractionTrace, error) {
	return s.interactions.GetWithTrace(ctx, logID)
}

// ListInteractions returns paginated interaction logs for the Compass dashboard.
func (s *Service) ListInteractions(ctx context.Context, limit, offset int32) ([]domain.InteractionLog, error) {
	return s.interactions.List(ctx, limit, offset)
}

// GetCustomerByID retrieves a customer profile by ID.
func (s *Service) GetCustomerByID(ctx context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	return s.customers.GetByID(ctx, id)
}

// UpdateCustomerTier updates the customer tier.
func (s *Service) UpdateCustomerTier(ctx context.Context, id uuid.UUID, tier domain.CustomerTier) error {
	return s.customers.UpdateTier(ctx, id, tier)
}

// ListInteractionsByCustomer retrieves all interactions for a specific customer.
func (s *Service) ListInteractionsByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.InteractionLog, error) {
	return s.interactions.ListByCustomer(ctx, customerID)
}

