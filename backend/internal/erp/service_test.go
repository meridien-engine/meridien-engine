package erp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/shopspring/decimal"
)

// ─── Mock repositories ────────────────────────────────────────────────────────

type mockProductRepo struct {
	products map[string]*domain.Product
}

func newMockProductRepo(products ...*domain.Product) *mockProductRepo {
	m := &mockProductRepo{products: make(map[string]*domain.Product)}
	for _, p := range products {
		m.products[p.SKU] = p
	}
	return m
}

func (m *mockProductRepo) GetBySKU(_ context.Context, sku string) (*domain.Product, error) {
	p, ok := m.products[sku]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (m *mockProductRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	for _, p := range m.products {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockProductRepo) List(_ context.Context) ([]domain.Product, error) {
	out := make([]domain.Product, 0, len(m.products))
	for _, p := range m.products {
		out = append(out, *p)
	}
	return out, nil
}

func (m *mockProductRepo) DecrementStock(_ context.Context, id uuid.UUID, qty int32) (*domain.Product, error) {
	for _, p := range m.products {
		if p.ID == id {
			if p.StockQty < qty {
				return nil, domain.ErrInsufficientStock
			}
			p.StockQty -= qty
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockProductRepo) Create(_ context.Context, p *domain.Product) (*domain.Product, error) {
	p.ID = uuid.New()
	m.products[p.SKU] = p
	return p, nil
}

type mockOrderRepo struct {
	orders []*domain.Order
}

func (m *mockOrderRepo) Create(_ context.Context, o *domain.Order) (*domain.Order, error) {
	m.orders = append(m.orders, o)
	return o, nil
}

func (m *mockOrderRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockOrderRepo) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]domain.Order, error) {
	var out []domain.Order
	for _, o := range m.orders {
		if o.CustomerID == customerID {
			out = append(out, *o)
		}
	}
	return out, nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.ID == id {
			o.Status = status
			return o, nil
		}
	}
	return nil, domain.ErrNotFound
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestPlaceOrder_Success(t *testing.T) {
	widget := &domain.Product{
		ID:       uuid.New(),
		SKU:      "WIDGET-01",
		Name:     "Premium Widget",
		Price:    decimal.NewFromFloat(19.99),
		StockQty: 50,
		IsActive: true,
	}

	productRepo := newMockProductRepo(widget)
	orderRepo := &mockOrderRepo{}
	svc := erp.NewService(productRepo, orderRepo)

	cmd := domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Notes:      "Test order",
		Lines: []domain.OrderLine{
			{SKU: "WIDGET-01", Quantity: 3},
		},
	}

	order, err := svc.PlaceOrder(context.Background(), cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Price should be resolved from catalog: 19.99 * 3 = 59.97
	expectedTotal := decimal.NewFromFloat(19.99).Mul(decimal.NewFromInt(3))
	if !order.TotalPrice.Equal(expectedTotal) {
		t.Errorf("expected total %s, got %s", expectedTotal, order.TotalPrice)
	}

	if order.Status != domain.OrderStatusPending {
		t.Errorf("expected status pending, got %s", order.Status)
	}

	if len(order.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(order.Items))
	}

	item := order.Items[0]
	if item.SKU != "WIDGET-01" {
		t.Errorf("expected SKU WIDGET-01, got %s", item.SKU)
	}
	if !item.UnitPrice.Equal(decimal.NewFromFloat(19.99)) {
		t.Errorf("expected unit price 19.99, got %s", item.UnitPrice)
	}
	if item.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", item.Quantity)
	}
}

func TestPlaceOrder_MultipleItems(t *testing.T) {
	widgetA := &domain.Product{
		ID: uuid.New(), SKU: "A", Name: "A", Price: decimal.NewFromFloat(10.00), StockQty: 100, IsActive: true,
	}
	widgetB := &domain.Product{
		ID: uuid.New(), SKU: "B", Name: "B", Price: decimal.NewFromFloat(25.50), StockQty: 50, IsActive: true,
	}

	productRepo := newMockProductRepo(widgetA, widgetB)
	orderRepo := &mockOrderRepo{}
	svc := erp.NewService(productRepo, orderRepo)

	cmd := domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourcePortal,
		Lines: []domain.OrderLine{
			{SKU: "A", Quantity: 2},  // 10.00 * 2 = 20.00
			{SKU: "B", Quantity: 1},  // 25.50 * 1 = 25.50
		},
	}

	order, err := svc.PlaceOrder(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTotal := decimal.NewFromFloat(45.50)
	if !order.TotalPrice.Equal(expectedTotal) {
		t.Errorf("expected total %s, got %s", expectedTotal, order.TotalPrice)
	}
	if len(order.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(order.Items))
	}
}

func TestPlaceOrder_EmptyLines_ReturnsError(t *testing.T) {
	svc := erp.NewService(newMockProductRepo(), &mockOrderRepo{})

	_, err := svc.PlaceOrder(context.Background(), domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Lines:      nil,
	})

	if err == nil {
		t.Fatal("expected error for empty lines")
	}
	if !errors.Is(err, domain.ErrInvalidOrder) {
		t.Errorf("expected ErrInvalidOrder, got %v", err)
	}
}

func TestPlaceOrder_ZeroQuantity_ReturnsError(t *testing.T) {
	widget := &domain.Product{
		ID: uuid.New(), SKU: "X", Name: "X", Price: decimal.NewFromFloat(5.00), StockQty: 10, IsActive: true,
	}
	svc := erp.NewService(newMockProductRepo(widget), &mockOrderRepo{})

	_, err := svc.PlaceOrder(context.Background(), domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Lines:      []domain.OrderLine{{SKU: "X", Quantity: 0}},
	})

	if err == nil {
		t.Fatal("expected error for zero quantity")
	}
	if !errors.Is(err, domain.ErrInvalidOrder) {
		t.Errorf("expected ErrInvalidOrder, got %v", err)
	}
}

func TestPlaceOrder_InvalidSKU_ReturnsError(t *testing.T) {
	svc := erp.NewService(newMockProductRepo(), &mockOrderRepo{})

	_, err := svc.PlaceOrder(context.Background(), domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Lines:      []domain.OrderLine{{SKU: "NONEXISTENT", Quantity: 1}},
	})

	if err == nil {
		t.Fatal("expected error for invalid SKU")
	}
	if !errors.Is(err, domain.ErrInvalidSKU) {
		t.Errorf("expected ErrInvalidSKU, got %v", err)
	}
}

func TestPlaceOrder_InsufficientStock_ReturnsError(t *testing.T) {
	widget := &domain.Product{
		ID: uuid.New(), SKU: "LOW", Name: "Low Stock", Price: decimal.NewFromFloat(9.99), StockQty: 2, IsActive: true,
	}
	svc := erp.NewService(newMockProductRepo(widget), &mockOrderRepo{})

	_, err := svc.PlaceOrder(context.Background(), domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Lines:      []domain.OrderLine{{SKU: "LOW", Quantity: 5}},
	})

	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestPlaceOrder_InactiveProduct_ReturnsError(t *testing.T) {
	widget := &domain.Product{
		ID: uuid.New(), SKU: "OFF", Name: "Discontinued", Price: decimal.NewFromFloat(1.00), StockQty: 100, IsActive: false,
	}
	svc := erp.NewService(newMockProductRepo(widget), &mockOrderRepo{})

	_, err := svc.PlaceOrder(context.Background(), domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Lines:      []domain.OrderLine{{SKU: "OFF", Quantity: 1}},
	})

	if err == nil {
		t.Fatal("expected error for inactive product")
	}
	if !errors.Is(err, domain.ErrInvalidSKU) {
		t.Errorf("expected ErrInvalidSKU, got %v", err)
	}
}

func TestPlaceOrder_PriceFromCatalog_NotFromAgent(t *testing.T) {
	// This is the CORE zero-hallucination test.
	// The agent has no price field in its command — price MUST come from the catalog.
	widget := &domain.Product{
		ID: uuid.New(), SKU: "PRICEY", Name: "Pricey Item", Price: decimal.NewFromFloat(99.99), StockQty: 10, IsActive: true,
	}
	svc := erp.NewService(newMockProductRepo(widget), &mockOrderRepo{})

	cmd := domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Lines:      []domain.OrderLine{{SKU: "PRICEY", Quantity: 1}},
		// Note: there is NO price field in OrderLine — by design.
	}

	order, err := svc.PlaceOrder(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Total must equal the catalog price, not any agent-supplied value.
	if !order.TotalPrice.Equal(decimal.NewFromFloat(99.99)) {
		t.Errorf("price must come from catalog (99.99), got %s", order.TotalPrice)
	}
	if !order.Items[0].UnitPrice.Equal(decimal.NewFromFloat(99.99)) {
		t.Errorf("unit price must be catalog price, got %s", order.Items[0].UnitPrice)
	}
}

func TestGetOrderStatus_NotFound(t *testing.T) {
	svc := erp.NewService(newMockProductRepo(), &mockOrderRepo{})

	_, err := svc.GetOrderStatus(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for nonexistent order")
	}
}
