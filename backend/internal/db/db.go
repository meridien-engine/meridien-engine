// Package db is the sqlc-generated database access layer.
//
// This file is a BUILD STUB — it exists so the repository layer compiles
// before sqlc generate has been run against a live database.
//
// Run `make sqlc` (or `sqlc generate` from the backend/ directory) after
// the Postgres container is healthy to overwrite this with real generated code.
//
// DO NOT hand-edit after sqlc has been run — all changes will be overwritten.
package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Queries ──────────────────────────────────────────────────────────────────

// Queries holds the compiled sqlc query set. Instantiated once in main.go.
type Queries struct {
	db DBTX
}

// DBTX is satisfied by both *sql.DB and *sql.Tx.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries          { return &Queries{db: db} }
func (q *Queries) WithTx(tx *sql.Tx) *Queries { return &Queries{db: tx} }

// ─── Row types (sqlc generates these from schema) ─────────────────────────────

type Product struct {
	ID          uuid.UUID
	BusinessID  uuid.UUID
	Sku         string
	Name        string
	Description sql.NullString
	Price       decimal.Decimal
	StockQty    int32
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   sql.NullTime
}

type Order struct {
	ID         uuid.UUID
	BusinessID uuid.UUID
	CustomerID uuid.UUID
	TotalPrice decimal.Decimal
	Status     string
	Source     string
	Notes      sql.NullString
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Sku       string
	Name      string
	Quantity  int32
	UnitPrice decimal.Decimal
}

type CustomerProfile struct {
	ID              uuid.UUID
	BusinessID      uuid.UUID
	UnifiedName     sql.NullString
	CustomerTier    string
	SemanticSummary sql.NullString
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CustomerChannel struct {
	ID                 uuid.UUID
	CustomerProfileID  uuid.UUID
	ChannelType        string
	ChannelExternalID  string
	CreatedAt          time.Time
}

type InteractionLog struct {
	ID          uuid.UUID
	BusinessID  uuid.UUID
	CustomerID  uuid.UUID
	Channel     string
	InboundMsg  string
	OutboundMsg string
	TokensUsed  int32
	LatencyMs   int32
	CreatedAt   time.Time
}

type InteractionTrace struct {
	ID                uuid.UUID
	InteractionLogID  uuid.UUID
	RetrievedContexts []byte
	SystemPrompt      sql.NullString
	RawAgentThoughts  sql.NullString
	ToolsCalled       []byte
	CreatedAt         time.Time
}

// ─── Param types ──────────────────────────────────────────────────────────────

type CreateProductParams struct {
	BusinessID  uuid.UUID
	Sku         string
	Name        string
	Description sql.NullString
	Price       decimal.Decimal
	StockQty    int32
}

type GetProductBySKUParams struct {
	BusinessID uuid.UUID
	Sku        string
}

type DecrementStockParams struct {
	ID       uuid.UUID
	Quantity int32
}

type CreateOrderParams struct {
	BusinessID uuid.UUID
	CustomerID uuid.UUID
	TotalPrice decimal.Decimal
	Status     string
	Source     string
	Notes      sql.NullString
}

type UpdateOrderStatusParams struct {
	ID     uuid.UUID
	Status string
}

type CreateOrderItemParams struct {
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Sku       string
	Name      string
	Quantity  int32
	UnitPrice decimal.Decimal
}

type CreateCustomerProfileParams struct {
	BusinessID   uuid.UUID
	UnifiedName  sql.NullString
	CustomerTier string
}

type UpsertCustomerChannelParams struct {
	CustomerProfileID uuid.UUID
	ChannelType       string
	ChannelExternalID string
}

type GetCustomerByChannelParams struct {
	ChannelType       string
	ChannelExternalID string
}

type UpdateSemanticSummaryParams struct {
	ID              uuid.UUID
	SemanticSummary sql.NullString
}

type UpdateCustomerTierParams struct {
	ID           uuid.UUID
	CustomerTier string
}

type CreateInteractionLogParams struct {
	BusinessID  uuid.UUID
	CustomerID  uuid.UUID
	Channel     string
	InboundMsg  string
	OutboundMsg string
	TokensUsed  int32
	LatencyMs   int32
}

type CreateInteractionTraceParams struct {
	InteractionLogID  uuid.UUID
	RetrievedContexts []byte
	SystemPrompt      string
	RawAgentThoughts  string
	ToolsCalled       []byte
}

type ListInteractionLogsParams struct {
	Lim int32
	Off int32
}

// GetInteractionWithTraceRow is the joined result for Compass detail view.
type GetInteractionWithTraceRow struct {
	ID                uuid.UUID
	BusinessID        uuid.UUID
	CustomerID        uuid.UUID
	Channel           string
	InboundMsg        string
	OutboundMsg       string
	TokensUsed        int32
	LatencyMs         int32
	CreatedAt         time.Time
	RetrievedContexts []byte
	SystemPrompt      sql.NullString
	RawAgentThoughts  sql.NullString
	ToolsCalled       []byte
}

// ─── Stub query methods (replaced by sqlc generate) ──────────────────────────

func (q *Queries) CreateProduct(_ context.Context, _ CreateProductParams) (Product, error) {
	panic("sqlc stub: run 'make sqlc' to generate real implementations")
}
func (q *Queries) GetProductBySKU(_ context.Context, _ GetProductBySKUParams) (Product, error) {
	panic("sqlc stub")
}
func (q *Queries) GetProductByID(_ context.Context, _ uuid.UUID) (Product, error) {
	panic("sqlc stub")
}
func (q *Queries) ListProducts(_ context.Context) ([]Product, error)      { panic("sqlc stub") }
func (q *Queries) DecrementStock(_ context.Context, _ DecrementStockParams) (Product, error) {
	panic("sqlc stub")
}
func (q *Queries) SoftDeleteProduct(_ context.Context, _ uuid.UUID) error { panic("sqlc stub") }

func (q *Queries) CreateOrder(_ context.Context, _ CreateOrderParams) (Order, error) {
	panic("sqlc stub")
}
func (q *Queries) GetOrderByID(_ context.Context, _ uuid.UUID) (Order, error) { panic("sqlc stub") }
func (q *Queries) ListOrdersByCustomer(_ context.Context, _ uuid.UUID) ([]Order, error) {
	panic("sqlc stub")
}
func (q *Queries) UpdateOrderStatus(_ context.Context, _ UpdateOrderStatusParams) (Order, error) {
	panic("sqlc stub")
}
func (q *Queries) CreateOrderItem(_ context.Context, _ CreateOrderItemParams) (OrderItem, error) {
	panic("sqlc stub")
}
func (q *Queries) ListOrderItems(_ context.Context, _ uuid.UUID) ([]OrderItem, error) {
	panic("sqlc stub")
}

func (q *Queries) CreateCustomerProfile(_ context.Context, _ CreateCustomerProfileParams) (CustomerProfile, error) {
	panic("sqlc stub")
}
func (q *Queries) GetCustomerProfileByID(_ context.Context, _ uuid.UUID) (CustomerProfile, error) {
	panic("sqlc stub")
}
func (q *Queries) UpsertCustomerChannel(_ context.Context, _ UpsertCustomerChannelParams) (CustomerChannel, error) {
	panic("sqlc stub")
}
func (q *Queries) GetCustomerByChannel(_ context.Context, _ GetCustomerByChannelParams) (CustomerProfile, error) {
	panic("sqlc stub")
}
func (q *Queries) UpdateSemanticSummary(_ context.Context, _ UpdateSemanticSummaryParams) (CustomerProfile, error) {
	panic("sqlc stub")
}
func (q *Queries) UpdateCustomerTier(_ context.Context, _ UpdateCustomerTierParams) (CustomerProfile, error) {
	panic("sqlc stub")
}

func (q *Queries) CreateInteractionLog(_ context.Context, _ CreateInteractionLogParams) (InteractionLog, error) {
	panic("sqlc stub")
}
func (q *Queries) CreateInteractionTrace(_ context.Context, _ CreateInteractionTraceParams) (InteractionTrace, error) {
	panic("sqlc stub")
}
func (q *Queries) GetInteractionWithTrace(_ context.Context, _ uuid.UUID) (GetInteractionWithTraceRow, error) {
	panic("sqlc stub")
}
func (q *Queries) ListInteractionLogs(_ context.Context, _ ListInteractionLogsParams) ([]InteractionLog, error) {
	panic("sqlc stub")
}
func (q *Queries) ListInteractionLogsByCustomer(_ context.Context, _ uuid.UUID) ([]InteractionLog, error) {
	panic("sqlc stub")
}
