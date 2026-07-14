// Package domain defines the pure business entities and value objects for
// the Meridien Engine. This package has zero external dependencies — no
// database drivers, no HTTP frameworks. It is the innermost ring of the
// onion architecture and the most stable layer of the codebase.
package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Product ──────────────────────────────────────────────────────────────────

type Product struct {
	ID          uuid.UUID
	BusinessID  uuid.UUID
	SKU         string
	Name        string
	Description string
	Price       decimal.Decimal
	StockQty    int32
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ─── Order ────────────────────────────────────────────────────────────────────

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type OrderSource string

const (
	OrderSourceAgent  OrderSource = "agent"
	OrderSourcePortal OrderSource = "portal"
)

type Order struct {
	ID         uuid.UUID
	BusinessID uuid.UUID
	CustomerID uuid.UUID
	TotalPrice decimal.Decimal
	Status     OrderStatus
	Source     OrderSource
	Notes      string
	Items      []OrderItem
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	SKU       string
	Name      string
	Quantity  int32
	UnitPrice decimal.Decimal // Catalog price sealed at checkout — never agent-supplied
}

// PlaceOrderCommand is the intent object passed from the ERP service handler
// into the order placement use-case. The agent submits SKU + Quantity only;
// price is resolved inside the use-case from the product catalog.
type PlaceOrderCommand struct {
	CustomerID uuid.UUID
	Source     OrderSource
	Notes      string
	Lines      []OrderLine
}

type OrderLine struct {
	SKU      string
	Quantity int32
}

// ─── Customer ────────────────────────────────────────────────────────────────

type CustomerTier string

const (
	TierStandard CustomerTier = "standard"
	TierSilver   CustomerTier = "silver"
	TierGold     CustomerTier = "gold"
)

type ChannelType string

const (
	ChannelWhatsApp  ChannelType = "whatsapp"
	ChannelMessenger ChannelType = "messenger"
	ChannelWeb       ChannelType = "web"
)

type CustomerProfile struct {
	ID              uuid.UUID
	BusinessID      uuid.UUID
	UnifiedName     string
	CustomerTier    CustomerTier
	SemanticSummary string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ─── Knowledge ────────────────────────────────────────────────────────────────

type KnowledgeChunk struct {
	NodeID  uuid.UUID
	Source  string
	Content string
	Score   float64 // Cosine similarity [0, 1]
}

// ─── Interaction ─────────────────────────────────────────────────────────────

type InteractionLog struct {
	ID          uuid.UUID
	BusinessID  uuid.UUID
	CustomerID  uuid.UUID
	Channel     ChannelType
	InboundMsg  string
	OutboundMsg string
	TokensUsed  int32
	LatencyMs   int32
	CreatedAt   time.Time
}

type RetrievedContext struct {
	Content string
	Score   float64
}

type ToolCall struct {
	ToolName   string
	ArgsJSON   string
	ResultJSON string
}

// HITLStatus represents the merchant review state for a suspended workflow.
type HITLStatus string

const (
	HITLStatusNone     HITLStatus = "none"
	HITLStatusPending  HITLStatus = "pending"
	HITLStatusApproved HITLStatus = "approved"
	HITLStatusRejected HITLStatus = "rejected"
	HITLStatusTimedOut HITLStatus = "timed_out"
)

type InteractionTrace struct {
	ID                uuid.UUID
	InteractionLogID  uuid.UUID
	RetrievedContexts []RetrievedContext
	SystemPrompt      string
	RawAgentThoughts  string
	ToolsCalled       []ToolCall
	// HITL suspension fields (zero-value = 'none', no suspension)
	WorkflowID  string
	HITLStatus  HITLStatus
	SuspendedAt *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
}
