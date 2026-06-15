package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/shopspring/decimal"
)

// ProductRepository implements domain.ProductRepository using sqlc + Postgres.
type ProductRepository struct {
	q  *db.Queries
	db *sql.DB
}

func NewProductRepository(database *sql.DB, q *db.Queries) *ProductRepository {
	return &ProductRepository{q: q, db: database}
}

func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business id: %w", err)
	}

	row, err := r.q.GetProductBySKU(ctx, db.GetProductBySKUParams{
		BusinessID: bid,
		Sku:        sku,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get product by sku: %w", err)
	}
	return mapProduct(row), nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	row, err := r.q.GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return mapProduct(row), nil
}

func (r *ProductRepository) List(ctx context.Context) ([]domain.Product, error) {
	rows, err := r.q.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	out := make([]domain.Product, len(rows))
	for i, row := range rows {
		out[i] = *mapProduct(row)
	}
	return out, nil
}

// DecrementStock atomically decrements stock using a conditional UPDATE.
// Returns ErrInsufficientStock if stock would go below zero.
func (r *ProductRepository) DecrementStock(ctx context.Context, id uuid.UUID, qty int32) (*domain.Product, error) {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var product *domain.Product
	err = ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		qtx := r.q.WithTx(tx)
		row, err := qtx.DecrementStock(ctx, db.DecrementStockParams{
			ID:       id,
			Quantity: qty,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrInsufficientStock
			}
			return fmt.Errorf("decrement stock: %w", err)
		}
		product = mapProduct(row)
		return nil
	})
	return product, err
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	businessID, err := BusinessIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var created *domain.Product
	err = ExecWithTenant(ctx, r.db, businessID, func(tx *sql.Tx) error {
		bid, _ := uuid.Parse(businessID)
		qtx := r.q.WithTx(tx)
		row, err := qtx.CreateProduct(ctx, db.CreateProductParams{
			BusinessID:  bid,
			Sku:         p.SKU,
			Name:        p.Name,
			Description: sql.NullString{String: p.Description, Valid: p.Description != ""},
			Price:       p.Price,
			StockQty:    p.StockQty,
		})
		if err != nil {
			return fmt.Errorf("create product: %w", err)
		}
		created = mapProduct(row)
		return nil
	})
	return created, err
}

// ─── mapper ───────────────────────────────────────────────────────────────────

func mapProduct(row db.Product) *domain.Product {
	return &domain.Product{
		ID:          row.ID,
		BusinessID:  row.BusinessID,
		SKU:         row.Sku,
		Name:        row.Name,
		Description: row.Description.String,
		Price:       row.Price,
		StockQty:    row.StockQty,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

// ensure decimal import is used (shopspring is used in mapProduct via sqlc type)
var _ = decimal.Zero
