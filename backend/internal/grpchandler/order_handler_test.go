package grpchandler_test

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/gen/orders"
	"github.com/meridien-engine/meridien-engine/internal/grpchandler"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// ─── gRPC Test Repositories ──────────────────────────────────────────────────

type mockGrpcProductRepo struct {
	products map[string]*domain.Product
}

func (m *mockGrpcProductRepo) GetBySKU(_ context.Context, sku string) (*domain.Product, error) {
	if p, ok := m.products[sku]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockGrpcProductRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	return nil, nil
}

func (m *mockGrpcProductRepo) List(_ context.Context) ([]domain.Product, error) {
	return nil, nil
}

func (m *mockGrpcProductRepo) DecrementStock(_ context.Context, id uuid.UUID, qty int32) (*domain.Product, error) {
	return nil, nil
}

func (m *mockGrpcProductRepo) Create(_ context.Context, p *domain.Product) (*domain.Product, error) {
	return nil, nil
}

type mockGrpcOrderRepo struct {
	orders map[uuid.UUID]*domain.Order
}

func (m *mockGrpcOrderRepo) Create(_ context.Context, o *domain.Order) (*domain.Order, error) {
	m.orders[o.ID] = o
	return o, nil
}

func (m *mockGrpcOrderRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Order, error) {
	if o, ok := m.orders[id]; ok {
		return o, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockGrpcOrderRepo) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]domain.Order, error) {
	var list []domain.Order
	for _, o := range m.orders {
		if o.CustomerID == customerID {
			list = append(list, *o)
		}
	}
	return list, nil
}

func (m *mockGrpcOrderRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	if o, ok := m.orders[id]; ok {
		o.Status = status
		return o, nil
	}
	return nil, domain.ErrNotFound
}

func setupBufConnServer(t *testing.T, handler *grpchandler.OrderHandler) *grpc.ClientConn {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	orders.RegisterOrderServiceServer(s, handler)

	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Errorf("Server exited with error: %v", err)
		}
	}()

	t.Cleanup(func() {
		s.GracefulStop()
	})

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

func TestOrderHandler_PlaceNewOrder_Success(t *testing.T) {
	pRepo := &mockGrpcProductRepo{
		products: map[string]*domain.Product{
			"SKU-ABC": {
				ID:       uuid.New(),
				SKU:      "SKU-ABC",
				Price:    decimal.NewFromInt(49),
				StockQty: 50,
				IsActive: true,
			},
		},
	}
	oRepo := &mockGrpcOrderRepo{orders: make(map[uuid.UUID]*domain.Order)}
	erpSvc := erp.NewService(pRepo, oRepo)
	handler := grpchandler.NewOrderHandler(erpSvc)

	conn := setupBufConnServer(t, handler)
	client := orders.NewOrderServiceClient(conn)

	req := &orders.OrderRequest{
		CustomerId: uuid.New().String(),
		Source:     "api",
		Notes:      "Urgent delivery",
		Items: []*orders.OrderLineItem{
			{
				Sku:      "SKU-ABC",
				Quantity: 2,
			},
		},
	}

	resp, err := client.PlaceNewOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no gRPC error, got: %v", err)
	}

	if resp.TotalPrice != 98.0 {
		t.Errorf("expected total price to be 98.0, got %f", resp.TotalPrice)
	}

	if resp.Status != string(domain.OrderStatusPending) {
		t.Errorf("expected order status pending, got %s", resp.Status)
	}
}

func TestOrderHandler_PlaceNewOrder_ValidationErrors(t *testing.T) {
	pRepo := &mockGrpcProductRepo{products: make(map[string]*domain.Product)}
	oRepo := &mockGrpcOrderRepo{orders: make(map[uuid.UUID]*domain.Order)}
	erpSvc := erp.NewService(pRepo, oRepo)
	handler := grpchandler.NewOrderHandler(erpSvc)

	conn := setupBufConnServer(t, handler)
	client := orders.NewOrderServiceClient(conn)

	tests := []struct {
		name string
		req  *orders.OrderRequest
		code codes.Code
	}{
		{
			name: "empty items list",
			req: &orders.OrderRequest{
				CustomerId: uuid.New().String(),
				Items:      []*orders.OrderLineItem{},
			},
			code: codes.InvalidArgument,
		},
		{
			name: "invalid customer ID",
			req: &orders.OrderRequest{
				CustomerId: "not-a-uuid",
				Items: []*orders.OrderLineItem{
					{Sku: "SKU-1", Quantity: 1},
				},
			},
			code: codes.InvalidArgument,
		},
		{
			name: "invalid quantity",
			req: &orders.OrderRequest{
				CustomerId: uuid.New().String(),
				Items: []*orders.OrderLineItem{
					{Sku: "SKU-1", Quantity: 0},
				},
			},
			code: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.PlaceNewOrder(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected gRPC error but got nil")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected status error, got: %v", err)
			}
			if st.Code() != tt.code {
				t.Errorf("expected status code %s, got %s", tt.code, st.Code())
			}
		})
	}
}

func TestOrderHandler_GetOrderStatus(t *testing.T) {
	pRepo := &mockGrpcProductRepo{products: make(map[string]*domain.Product)}
	orderID := uuid.New()
	oRepo := &mockGrpcOrderRepo{
		orders: map[uuid.UUID]*domain.Order{
			orderID: {
				ID:         orderID,
				CustomerID: uuid.New(),
				Status:     domain.OrderStatusPending,
				TotalPrice: decimal.NewFromInt(50),
			},
		},
	}
	erpSvc := erp.NewService(pRepo, oRepo)
	handler := grpchandler.NewOrderHandler(erpSvc)

	conn := setupBufConnServer(t, handler)
	client := orders.NewOrderServiceClient(conn)

	// Status query
	resp, err := client.GetOrderStatus(context.Background(), &orders.StatusRequest{OrderId: orderID.String()})
	if err != nil {
		t.Fatalf("expected no query error, got %v", err)
	}

	if resp.Status != string(domain.OrderStatusPending) {
		t.Errorf("expected status %s, got %s", domain.OrderStatusPending, resp.Status)
	}

	// Not found query
	_, err = client.GetOrderStatus(context.Background(), &orders.StatusRequest{OrderId: uuid.New().String()})
	if err == nil {
		t.Fatal("expected error for non-existent order")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected codes.NotFound, got %s", st.Code())
	}
}
