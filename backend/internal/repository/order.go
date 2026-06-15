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

// OrderRepository implements domain.OrderRepository using sqlc + Postgres.
type OrderRepository struct {
	q  *db.Queries
	db *sql.DB
}

func NewOrderRepository(database *sql.DB, q *db.Queries) *OrderRepository {
	return &OrderRepository{q: q, db: database}
}

// Create inserts the order header and all line items inside a single
// RLS-scoped transaction. Stock has already been validated by the service
// layer; this method only writes — it does NOT re-validate stock.
func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) (*domain.Order, error) {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business id: %w", err)
	}

	err = ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		qtx := r.q.WithTx(tx)

		// Insert order header.
		row, err := qtx.CreateOrder(ctx, db.CreateOrderParams{
			BusinessID: bid,
			CustomerID: o.CustomerID,
			TotalPrice: o.TotalPrice,
			Status:     string(o.Status),
			Source:     string(o.Source),
			Notes:      sql.NullString{String: o.Notes, Valid: o.Notes != ""},
		})
		if err != nil {
			return fmt.Errorf("create order header: %w", err)
		}
		o.ID = row.ID
		o.CreatedAt = row.CreatedAt
		o.UpdatedAt = row.UpdatedAt

		// Insert line items.
		for i := range o.Items {
			item := &o.Items[i]
			itemRow, err := qtx.CreateOrderItem(ctx, db.CreateOrderItemParams{
				OrderID:   o.ID,
				ProductID: item.ProductID,
				Sku:       item.SKU,
				Name:      item.Name,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
			})
			if err != nil {
				return fmt.Errorf("create order item (sku=%s): %w", item.SKU, err)
			}
			item.ID = itemRow.ID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	row, err := r.q.GetOrderByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	items, err := r.q.ListOrderItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}

	return mapOrder(row, items), nil
}

func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.Order, error) {
	rows, err := r.q.ListOrdersByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list orders by customer: %w", err)
	}
	out := make([]domain.Order, len(rows))
	for i, row := range rows {
		out[i] = *mapOrder(row, nil) // items not loaded in list view
	}
	return out, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var updated *domain.Order
	err = ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		qtx := r.q.WithTx(tx)
		row, err := qtx.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
			ID:     id,
			Status: string(status),
		})
		if err != nil {
			return fmt.Errorf("update order status: %w", err)
		}
		updated = mapOrder(row, nil)
		return nil
	})
	return updated, err
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func mapOrder(row db.Order, items []db.OrderItem) *domain.Order {
	o := &domain.Order{
		ID:         row.ID,
		BusinessID: row.BusinessID,
		CustomerID: row.CustomerID,
		TotalPrice: row.TotalPrice,
		Status:     domain.OrderStatus(row.Status),
		Source:     domain.OrderSource(row.Source),
		Notes:      row.Notes.String,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		Items:      make([]domain.OrderItem, len(items)),
	}
	for i, it := range items {
		o.Items[i] = domain.OrderItem{
			ID:        it.ID,
			OrderID:   it.OrderID,
			ProductID: it.ProductID,
			SKU:       it.Sku,
			Name:      it.Name,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		}
	}
	return o
}
