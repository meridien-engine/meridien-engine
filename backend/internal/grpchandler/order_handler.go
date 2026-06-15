// Package grpchandler implements the gRPC server handlers for the ERP OrderService.
package grpchandler

import (
	"context"
	"errors"

	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"github.com/meridien-engine/meridien-engine/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
)

// OrderHandler implements the gRPC OrderService server interface.
// It bridges the transport layer to the ERP application service.
type OrderHandler struct {
	svc *erp.Service
}

func NewOrderHandler(svc *erp.Service) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// PlaceNewOrder is the gRPC handler for order creation.
// Business context is extracted from the authenticated JWT claim stored
// in the incoming metadata, injected into context via the auth interceptor.
func (h *OrderHandler) PlaceNewOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
	// Validate required fields.
	if len(req.Items) == 0 {
		metrics.OrderValidationErrors.WithLabelValues("invalid_order").Inc()
		return nil, status.Error(codes.InvalidArgument, "order must contain at least one item")
	}

	customerID, err := uuid.Parse(req.CustomerID)
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

	return &PlaceOrderResponse{
		OrderID:        order.ID.String(),
		Status:         string(order.Status),
		ConfirmationMsg: "Order placed successfully. Total: " + order.TotalPrice.String(),
		TotalPrice:     order.TotalPrice.InexactFloat64(),
	}, nil
}

func (h *OrderHandler) GetOrderStatus(ctx context.Context, req *GetOrderStatusRequest) (*GetOrderStatusResponse, error) {
	id, err := uuid.Parse(req.OrderID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	order, err := h.svc.GetOrderStatus(ctx, id)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &GetOrderStatusResponse{
		OrderID: order.ID.String(),
		Status:  string(order.Status),
	}, nil
}

func (h *OrderHandler) GetOrderDetails(ctx context.Context, req *GetOrderDetailsRequest) (*GetOrderDetailsResponse, error) {
	id, err := uuid.Parse(req.OrderID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	order, err := h.svc.GetOrderDetails(ctx, id)
	if err != nil {
		return nil, mapDomainError(err)
	}

	items := make([]*OrderLineItemDetail, len(order.Items))
	for i, it := range order.Items {
		items[i] = &OrderLineItemDetail{
			Sku:       it.SKU,
			Name:      it.Name,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice.InexactFloat64(),
		}
	}

	return &GetOrderDetailsResponse{
		OrderID: order.ID.String(),
		Status:  string(order.Status),
		Total:   order.TotalPrice.InexactFloat64(),
		Items:   items,
	}, nil
}

// ─── Inline DTO types (replace with generated protobuf types after proto compile) ──

type PlaceOrderRequest struct {
	CustomerID string
	Source     string
	Notes      string
	Items      []*OrderLineItem
}
type OrderLineItem struct {
	Sku      string
	Quantity int32
}
type PlaceOrderResponse struct {
	OrderID         string
	Status          string
	ConfirmationMsg string
	TotalPrice      float64
}
type GetOrderStatusRequest struct{ OrderID string }
type GetOrderStatusResponse struct {
	OrderID           string
	Status            string
	EstimatedDelivery string
}
type GetOrderDetailsRequest struct{ OrderID string }
type OrderLineItemDetail struct {
	Sku       string
	Name      string
	Quantity  int32
	UnitPrice float64
}
type GetOrderDetailsResponse struct {
	OrderID string
	Status  string
	Total   float64
	Items   []*OrderLineItemDetail
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

// ensure repository import used via ExecWithTenant in tests.
var _ = repository.WithBusinessID
