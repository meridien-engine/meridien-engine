// Package erp implements the ERP service layer — the transactional core
// of Meridien Engine. It orchestrates product catalog lookups, stock
// decrements, and order creation inside atomic database transactions.
package erp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/shopspring/decimal"
)

// Service is the ERP application service. It depends only on domain
// repository interfaces — never directly on database drivers.
type Service struct {
	products domain.ProductRepository
	orders   domain.OrderRepository
}

func NewService(products domain.ProductRepository, orders domain.OrderRepository) *Service {
	return &Service{products: products, orders: orders}
}

// PlaceOrder executes the full order placement flow:
//  1. Validates the command (non-empty lines, positive quantities).
//  2. For each line: resolves the product by SKU, checks stock.
//  3. Atomically decrements stock and creates the order + items.
//
// Price is always taken from the database catalog (product.Price).
// The AI agent has no mechanism to influence pricing.
func (s *Service) PlaceOrder(ctx context.Context, cmd domain.PlaceOrderCommand) (*domain.Order, error) {
	if len(cmd.Lines) == 0 {
		return nil, fmt.Errorf("%w: order must have at least one line item", domain.ErrInvalidOrder)
	}

	// ── Phase 1: Validate & resolve all line items ────────────────────────────
	// We do this before touching stock to fail fast on bad SKUs.
	type resolvedLine struct {
		product *domain.Product
		qty     int32
	}

	resolved := make([]resolvedLine, 0, len(cmd.Lines))

	for _, line := range cmd.Lines {
		if line.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity must be > 0 for SKU %q", domain.ErrInvalidOrder, line.SKU)
		}

		p, err := s.products.GetBySKU(ctx, line.SKU)
		if err != nil {
			return nil, fmt.Errorf("%w: SKU %q not found in catalog", domain.ErrInvalidSKU, line.SKU)
		}

		if !p.IsActive {
			return nil, fmt.Errorf("%w: product %q is not available", domain.ErrInvalidSKU, line.SKU)
		}

		if int32(p.StockQty) < line.Quantity {
			return nil, fmt.Errorf("%w: requested %d of %q but only %d in stock",
				domain.ErrInsufficientStock, line.Quantity, line.SKU, p.StockQty)
		}

		resolved = append(resolved, resolvedLine{product: p, qty: line.Quantity})
	}

	// ── Phase 2: Build Order domain object ────────────────────────────────────
	// Total price is computed from catalog prices — zero-hallucination guarantee.
	order := &domain.Order{
		ID:         uuid.New(),
		CustomerID: cmd.CustomerID,
		Source:     cmd.Source,
		Notes:      cmd.Notes,
		Status:     domain.OrderStatusPending,
		Items:      make([]domain.OrderItem, 0, len(resolved)),
	}

	for _, r := range resolved {
		lineTotal := r.product.Price.Mul(decimal.NewFromInt(int64(r.qty)))
		order.TotalPrice = order.TotalPrice.Add(lineTotal)
		order.Items = append(order.Items, domain.OrderItem{
			ID:        uuid.New(),
			ProductID: r.product.ID,
			SKU:       r.product.SKU,
			Name:      r.product.Name,
			Quantity:  r.qty,
			UnitPrice: r.product.Price, // Sealed from catalog
		})
	}

	// ── Phase 3: Persist (stock decrement + order creation in one tx) ─────────
	// The repository implementation wraps both operations in ExecWithTenant.
	created, err := s.orders.Create(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	return created, nil
}

// GetOrderStatus returns the current status of an order.
func (s *Service) GetOrderStatus(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

// GetOrderDetails returns the full order with line items.
func (s *Service) GetOrderDetails(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	return s.orders.GetByID(ctx, orderID)
}

// CreateProduct creates a new product catalog entry.
func (s *Service) CreateProduct(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	return s.products.Create(ctx, p)
}

// GetProductByID retrieves a product by its ID.
func (s *Service) GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return s.products.GetByID(ctx, id)
}

// ListProducts retrieves all active products.
func (s *Service) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.products.List(ctx)
}

// ListOrdersByCustomer retrieves all orders placed by a specific customer.
func (s *Service) ListOrdersByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.Order, error) {
	return s.orders.ListByCustomer(ctx, customerID)
}

// UpdateOrderStatus updates the status of an existing order.
func (s *Service) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	return s.orders.UpdateStatus(ctx, orderID, status)
}

