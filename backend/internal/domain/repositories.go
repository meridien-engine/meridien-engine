// Package domain — Repository interfaces.
//
// These interfaces define the data access contract for each aggregate.
// The domain layer owns them; infrastructure (internal/repository) implements
// them. This inversion allows the domain and service layers to be tested
// with mock implementations without touching the database.
package domain

import (
	"context"

	"github.com/google/uuid"
)

// ─── Product Repository ───────────────────────────────────────────────────────

type ProductRepository interface {
	// GetBySKU returns a product for the active tenant by SKU.
	GetBySKU(ctx context.Context, sku string) (*Product, error)

	// GetByID returns a product for the active tenant by primary key.
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)

	// List returns all active products for the active tenant.
	List(ctx context.Context) ([]Product, error)

	// DecrementStock atomically reduces stock_qty by the given amount.
	// Returns ErrInsufficientStock if stock would go below zero.
	DecrementStock(ctx context.Context, id uuid.UUID, qty int32) (*Product, error)

	// Create inserts a new product and returns the persisted record.
	Create(ctx context.Context, p *Product) (*Product, error)
}

// ─── Order Repository ─────────────────────────────────────────────────────────

type OrderRepository interface {
	// Create inserts an order and its items inside a single transaction.
	// The RLS context must be set by the caller before invoking.
	Create(ctx context.Context, o *Order) (*Order, error)

	// GetByID returns an order with its line items.
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)

	// ListByCustomer returns all orders for a given customer.
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]Order, error)

	// UpdateStatus transitions an order to a new status.
	UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) (*Order, error)
}

// ─── Customer Repository ──────────────────────────────────────────────────────

type CustomerRepository interface {
	// GetByChannel resolves a UCM record from a channel type + external ID.
	// Returns the profile and a boolean indicating whether it was newly created.
	GetOrCreateByChannel(ctx context.Context, channel ChannelType, externalID string) (*CustomerProfile, bool, error)

	// GetByID returns a profile by its primary key.
	GetByID(ctx context.Context, id uuid.UUID) (*CustomerProfile, error)

	// UpdateSemanticSummary overwrites the rolling semantic summary.
	UpdateSemanticSummary(ctx context.Context, id uuid.UUID, summary string) error

	// UpdateTier changes the customer loyalty tier.
	UpdateTier(ctx context.Context, id uuid.UUID, tier CustomerTier) error
}

// ─── Knowledge Repository ─────────────────────────────────────────────────────

type KnowledgeRepository interface {
	// Query performs a cosine similarity search and returns the top-k chunks.
	Query(ctx context.Context, queryEmbedding []float32, topK int) ([]KnowledgeChunk, error)

	// InsertChunk persists a single pre-embedded knowledge chunk.
	InsertChunk(ctx context.Context, source, content string, embedding []float32) error
}

// ─── Interaction Repository ───────────────────────────────────────────────────

type InteractionRepository interface {
	// RecordTurn persists the interaction log and its associated trace atomically.
	RecordTurn(ctx context.Context, log *InteractionLog, trace *InteractionTrace) error

	// GetWithTrace fetches a log and its full trace by log ID.
	GetWithTrace(ctx context.Context, id uuid.UUID) (*InteractionLog, *InteractionTrace, error)

	// List returns paginated interaction logs for Compass.
	List(ctx context.Context, limit, offset int32) ([]InteractionLog, error)

	// ListByCustomer returns all logs for a given customer.
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]InteractionLog, error)
}
