package agent_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/mera/agent"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	"github.com/shopspring/decimal"
)

// ─── Mock Implementations ───────────────────────────────────────────────────

type mockProductRepo struct{}

func (m *mockProductRepo) GetBySKU(_ context.Context, sku string) (*domain.Product, error) {
	return &domain.Product{
		ID:         uuid.New(),
		BusinessID: uuid.New(),
		SKU:        sku,
		Name:       "Test Product",
		Price:      decimal.NewFromInt(100),
		StockQty:   10,
		IsActive:   true,
	}, nil
}

func (m *mockProductRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	return nil, nil
}

func (m *mockProductRepo) List(_ context.Context) ([]domain.Product, error) {
	return nil, nil
}

func (m *mockProductRepo) DecrementStock(_ context.Context, id uuid.UUID, qty int32) (*domain.Product, error) {
	return nil, nil
}

func (m *mockProductRepo) Create(_ context.Context, p *domain.Product) (*domain.Product, error) {
	return nil, nil
}

type mockOrderRepo struct{}

func (m *mockOrderRepo) Create(_ context.Context, o *domain.Order) (*domain.Order, error) {
	return o, nil
}

func (m *mockOrderRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Order, error) {
	return nil, nil
}

func (m *mockOrderRepo) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]domain.Order, error) {
	return nil, nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	return nil, nil
}

type mockCustomerRepo struct{}

func (m *mockCustomerRepo) GetOrCreateByChannel(_ context.Context, channel domain.ChannelType, externalID string) (*domain.CustomerProfile, bool, error) {
	return &domain.CustomerProfile{
		ID:           uuid.New(),
		BusinessID:   uuid.New(),
		CustomerTier: domain.TierStandard,
	}, true, nil
}

func (m *mockCustomerRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	return nil, nil
}

func (m *mockCustomerRepo) UpdateSemanticSummary(_ context.Context, id uuid.UUID, summary string) error {
	return nil
}

func (m *mockCustomerRepo) UpdateTier(_ context.Context, id uuid.UUID, tier domain.CustomerTier) error {
	return nil
}

type mockInteractionRepo struct{}

func (m *mockInteractionRepo) RecordTurn(_ context.Context, log *domain.InteractionLog, trace *domain.InteractionTrace) error {
	return nil
}

func (m *mockInteractionRepo) GetWithTrace(_ context.Context, id uuid.UUID) (*domain.InteractionLog, *domain.InteractionTrace, error) {
	return nil, nil, nil
}

func (m *mockInteractionRepo) List(_ context.Context, limit, offset int32) ([]domain.InteractionLog, error) {
	return nil, nil
}

func (m *mockInteractionRepo) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]domain.InteractionLog, error) {
	return nil, nil
}

type mockKnowledgeRepo struct{}

func (m *mockKnowledgeRepo) Query(_ context.Context, queryEmbedding []float32, topK int) ([]domain.KnowledgeChunk, error) {
	return []domain.KnowledgeChunk{
		{
			NodeID:  uuid.New(),
			Source:  "faq.txt",
			Content: "This is test RAG content",
			Score:   0.9,
		},
	}, nil
}

func (m *mockKnowledgeRepo) InsertChunk(_ context.Context, source, content string, embedding []float32) error {
	return nil
}

func TestNewMeraWorkflow_Construction(t *testing.T) {
	pRepo := &mockProductRepo{}
	oRepo := &mockOrderRepo{}
	cRepo := &mockCustomerRepo{}
	iRepo := &mockInteractionRepo{}
	kRepo := &mockKnowledgeRepo{}

	synSvc := synapse.NewService(cRepo, iRepo)
	erpSvc := erp.NewService(pRepo, oRepo)

	wfAgent, err := agent.NewMeraWorkflow(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo)
	if err != nil {
		t.Fatalf("expected no workflow construction error, got %v", err)
	}

	if wfAgent == nil {
		t.Fatal("expected constructed agent to not be nil")
	}

	if wfAgent.Name() != "mera_workflow" {
		t.Errorf("expected workflow name 'mera_workflow', got '%s'", wfAgent.Name())
	}
}
