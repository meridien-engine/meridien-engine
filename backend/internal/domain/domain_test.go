package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/shopspring/decimal"
)

// ─── Entity construction tests ────────────────────────────────────────────────

func TestProduct_FieldsInitialise(t *testing.T) {
	p := domain.Product{
		ID:         uuid.New(),
		BusinessID: uuid.New(),
		SKU:        "SKU-001",
		Name:       "Test Widget",
		Price:      decimal.NewFromFloat(29.99),
		StockQty:   100,
		IsActive:   true,
	}

	if p.SKU != "SKU-001" {
		t.Errorf("expected SKU SKU-001, got %s", p.SKU)
	}
	if !p.Price.Equal(decimal.NewFromFloat(29.99)) {
		t.Errorf("expected price 29.99, got %s", p.Price.String())
	}
	if p.StockQty != 100 {
		t.Errorf("expected stock 100, got %d", p.StockQty)
	}
}

func TestOrderStatus_Constants(t *testing.T) {
	tests := []struct {
		status domain.OrderStatus
		want   string
	}{
		{domain.OrderStatusPending, "pending"},
		{domain.OrderStatusCompleted, "completed"},
		{domain.OrderStatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("OrderStatus %q != %q", tt.status, tt.want)
		}
	}
}

func TestOrderSource_Constants(t *testing.T) {
	if string(domain.OrderSourceAgent) != "agent" {
		t.Error("OrderSourceAgent should be 'agent'")
	}
	if string(domain.OrderSourcePortal) != "portal" {
		t.Error("OrderSourcePortal should be 'portal'")
	}
}

func TestCustomerTier_Constants(t *testing.T) {
	tiers := []struct {
		tier domain.CustomerTier
		want string
	}{
		{domain.TierStandard, "standard"},
		{domain.TierSilver, "silver"},
		{domain.TierGold, "gold"},
	}
	for _, tt := range tiers {
		if string(tt.tier) != tt.want {
			t.Errorf("CustomerTier %q != %q", tt.tier, tt.want)
		}
	}
}

func TestChannelType_Constants(t *testing.T) {
	channels := []struct {
		ch   domain.ChannelType
		want string
	}{
		{domain.ChannelWhatsApp, "whatsapp"},
		{domain.ChannelMessenger, "messenger"},
		{domain.ChannelWeb, "web"},
	}
	for _, tt := range channels {
		if string(tt.ch) != tt.want {
			t.Errorf("ChannelType %q != %q", tt.ch, tt.want)
		}
	}
}

func TestPlaceOrderCommand_EmptyLines(t *testing.T) {
	cmd := domain.PlaceOrderCommand{
		CustomerID: uuid.New(),
		Source:     domain.OrderSourceAgent,
		Lines:      nil,
	}
	if len(cmd.Lines) != 0 {
		t.Error("expected empty lines slice")
	}
}

// ─── Error sentinel tests ─────────────────────────────────────────────────────

func TestErrors_AreDistinct(t *testing.T) {
	errs := []error{
		domain.ErrNotFound,
		domain.ErrInsufficientStock,
		domain.ErrInvalidSKU,
		domain.ErrInvalidOrder,
		domain.ErrUnauthorised,
	}

	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errs[i] == errs[j] {
				t.Errorf("sentinel errors %d and %d should be distinct", i, j)
			}
		}
	}
}

func TestErrors_HaveMessages(t *testing.T) {
	if domain.ErrNotFound.Error() == "" {
		t.Error("ErrNotFound should have a message")
	}
	if domain.ErrInsufficientStock.Error() == "" {
		t.Error("ErrInsufficientStock should have a message")
	}
}
