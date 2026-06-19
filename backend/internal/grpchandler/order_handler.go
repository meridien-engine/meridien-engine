// Package grpchandler implements the gRPC server handlers for the ERP OrderService.
package grpchandler

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/gen/orders"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderHandler implements the gRPC OrderService server interface.
// It bridges the transport layer to the ERP application service.
type OrderHandler struct {
	orders.UnimplementedOrderServiceServer
	svc *erp.Service
}

func NewOrderHandler(svc *erp.Service) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// PlaceNewOrder is the gRPC handler for order creation.
// Business context is extracted from the authenticated JWT claim stored
// in the incoming metadata, injected into context via the auth interceptor.
func (h *OrderHandler) PlaceNewOrder(ctx context.Context, req *orders.OrderRequest) (*orders.OrderResponse, error) {
	// Validate required fields.
	if len(req.Items) == 0 {
		metrics.OrderValidationErrors.WithLabelValues("invalid_order").Inc()
		return nil, status.Error(codes.InvalidArgument, "order must contain at least one item")
	}

	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id: must be a UUID")
	}

	// Build the domain command — no price fields, by design.
	lines := make([]domain.OrderLine, len(req.Items))
	for i, item := range req.Items {
		if item.Quantity <= 0 {
			metrics.OrderValidationErrors.WithLabelValues("invalid_order").Inc()
			return nil, status.Errorf(codes.InvalidArgument, "quantity must be > 0 for SKU %q", item.Sku)
		}
		lines[i] = domain.OrderLine{SKU: item.Sku, Quantity: item.Quantity}
	}

	cmd := domain.PlaceOrderCommand{
		CustomerID: customerID,
		Source:     domain.OrderSource(req.Source),
		Notes:      req.Notes,
		Lines:      lines,
	}

	order, err := h.svc.PlaceOrder(ctx, cmd)
	if err != nil {
		return nil, mapDomainError(err)
	}

	metrics.OrdersPlacedTotal.WithLabelValues(string(order.Source)).Inc()

	return &orders.OrderResponse{
		OrderId:         order.ID.String(),
		Status:          string(order.Status),
		ConfirmationMsg: "Order placed successfully. Total: " + order.TotalPrice.String(),
		TotalPrice:      order.TotalPrice.InexactFloat64(),
	}, nil
}

func (h *OrderHandler) GetOrderStatus(ctx context.Context, req *orders.StatusRequest) (*orders.StatusResponse, error) {
	id, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	order, err := h.svc.GetOrderStatus(ctx, id)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &orders.StatusResponse{
		OrderId: order.ID.String(),
		Status:  string(order.Status),
	}, nil
}

func (h *OrderHandler) GetOrderDetails(ctx context.Context, req *orders.DetailsRequest) (*orders.DetailsResponse, error) {
	id, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	order, err := h.svc.GetOrderDetails(ctx, id)
	if err != nil {
		return nil, mapDomainError(err)
	}

	items := make([]*orders.OrderLineItemDetail, len(order.Items))
	for i, it := range order.Items {
		items[i] = &orders.OrderLineItemDetail{
			Sku:       it.SKU,
			Name:      it.Name,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice.InexactFloat64(),
		}
	}

	return &orders.DetailsResponse{
		OrderId: order.ID.String(),
		Status:  string(order.Status),
		Total:   order.TotalPrice.InexactFloat64(),
		Items:   items,
	}, nil
}

// CreateProduct creates a new product catalog entry.
func (h *OrderHandler) CreateProduct(ctx context.Context, req *orders.CreateProductRequest) (*orders.CreateProductResponse, error) {
	if req.Sku == "" || req.Name == "" || req.Price < 0 || req.StockQty < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid product payload")
	}
	p := &domain.Product{
		SKU:         req.Sku,
		Name:        req.Name,
		Description: req.Description,
		Price:       decimal.NewFromFloat(req.Price),
		StockQty:    req.StockQty,
	}
	created, err := h.svc.CreateProduct(ctx, p)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &orders.CreateProductResponse{
		Product: &orders.ProductDetail{
			Id:          created.ID.String(),
			Sku:         created.SKU,
			Name:        created.Name,
			Description: created.Description,
			Price:       created.Price.InexactFloat64(),
			StockQty:    created.StockQty,
			IsActive:    created.IsActive,
		},
	}, nil
}

// GetProductByID retrieves a product by its ID.
func (h *OrderHandler) GetProductByID(ctx context.Context, req *orders.GetProductByIDRequest) (*orders.GetProductByIDResponse, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id UUID")
	}
	product, err := h.svc.GetProductByID(ctx, id)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &orders.GetProductByIDResponse{
		Product: &orders.ProductDetail{
			Id:          product.ID.String(),
			Sku:         product.SKU,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price.InexactFloat64(),
			StockQty:    product.StockQty,
			IsActive:    product.IsActive,
		},
	}, nil
}

// ListProducts retrieves all active products.
func (h *OrderHandler) ListProducts(ctx context.Context, req *orders.ListProductsRequest) (*orders.ListProductsResponse, error) {
	products, err := h.svc.ListProducts(ctx)
	if err != nil {
		return nil, mapDomainError(err)
	}
	resp := &orders.ListProductsResponse{
		Products: make([]*orders.ProductDetail, len(products)),
	}
	for i, p := range products {
		resp.Products[i] = &orders.ProductDetail{
			Id:          p.ID.String(),
			Sku:         p.SKU,
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price.InexactFloat64(),
			StockQty:    p.StockQty,
			IsActive:    p.IsActive,
		}
	}
	return resp, nil
}

// ListOrdersByCustomer retrieves all orders placed by a specific customer.
func (h *OrderHandler) ListOrdersByCustomer(ctx context.Context, req *orders.ListOrdersByCustomerRequest) (*orders.ListOrdersByCustomerResponse, error) {
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id UUID")
	}
	ordersList, err := h.svc.ListOrdersByCustomer(ctx, customerID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	resp := &orders.ListOrdersByCustomerResponse{
		Orders: make([]*orders.OrderSummary, len(ordersList)),
	}
	for i, o := range ordersList {
		resp.Orders[i] = &orders.OrderSummary{
			OrderId:    o.ID.String(),
			CustomerId: o.CustomerID.String(),
			TotalPrice: o.TotalPrice.InexactFloat64(),
			Status:      string(o.Status),
			Source:      string(o.Source),
			Notes:       o.Notes,
			CreatedAt:   o.CreatedAt.Format(time.RFC3339),
		}
	}
	return resp, nil
}

// UpdateOrderStatus updates the status of an existing order.
func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *orders.UpdateOrderStatusRequest) (*orders.UpdateOrderStatusResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id UUID")
	}
	o, err := h.svc.UpdateOrderStatus(ctx, orderID, domain.OrderStatus(req.Status))
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &orders.UpdateOrderStatusResponse{
		Order: &orders.OrderSummary{
			OrderId:    o.ID.String(),
			CustomerId: o.CustomerID.String(),
			TotalPrice: o.TotalPrice.InexactFloat64(),
			Status:      string(o.Status),
			Source:      string(o.Source),
			Notes:       o.Notes,
			CreatedAt:   o.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// ─── Error mapping ────────────────────────────────────────────────────────────

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInsufficientStock):
		metrics.OrderValidationErrors.WithLabelValues("insufficient_stock").Inc()
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidSKU):
		metrics.OrderValidationErrors.WithLabelValues("invalid_sku").Inc()
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidOrder):
		metrics.OrderValidationErrors.WithLabelValues("invalid_order").Inc()
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrUnauthorised):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

