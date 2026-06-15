package synapse_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
)

// ─── Mock repositories ────────────────────────────────────────────────────────

type mockCustomerRepo struct {
	profiles map[string]*domain.CustomerProfile // keyed by channelType:externalID
}

func newMockCustomerRepo() *mockCustomerRepo {
	return &mockCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
}

func (m *mockCustomerRepo) GetOrCreateByChannel(
	_ context.Context,
	channel domain.ChannelType,
	externalID string,
) (*domain.CustomerProfile, bool, error) {
	key := string(channel) + ":" + externalID
	if p, ok := m.profiles[key]; ok {
		return p, false, nil
	}
	p := &domain.CustomerProfile{
		ID:           uuid.New(),
		BusinessID:   uuid.New(),
		CustomerTier: domain.TierStandard,
	}
	m.profiles[key] = p
	return p, true, nil
}

func (m *mockCustomerRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	for _, p := range m.profiles {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockCustomerRepo) UpdateSemanticSummary(_ context.Context, id uuid.UUID, summary string) error {
	for _, p := range m.profiles {
		if p.ID == id {
			p.SemanticSummary = summary
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockCustomerRepo) UpdateTier(_ context.Context, id uuid.UUID, tier domain.CustomerTier) error {
	for _, p := range m.profiles {
		if p.ID == id {
			p.CustomerTier = tier
			return nil
		}
	}
	return domain.ErrNotFound
}

type mockInteractionRepo struct {
	logs   []*domain.InteractionLog
	traces []*domain.InteractionTrace
}

func (m *mockInteractionRepo) RecordTurn(_ context.Context, log *domain.InteractionLog, trace *domain.InteractionTrace) error {
	m.logs = append(m.logs, log)
	m.traces = append(m.traces, trace)
	return nil
}

func (m *mockInteractionRepo) GetWithTrace(_ context.Context, id uuid.UUID) (*domain.InteractionLog, *domain.InteractionTrace, error) {
	for i, log := range m.logs {
		if log.ID == id {
			return log, m.traces[i], nil
		}
	}
	return nil, nil, domain.ErrNotFound
}

func (m *mockInteractionRepo) List(_ context.Context, limit, offset int32) ([]domain.InteractionLog, error) {
	start := int(offset)
	end := start + int(limit)
	if start >= len(m.logs) {
		return nil, nil
	}
	if end > len(m.logs) {
		end = len(m.logs)
	}
	out := make([]domain.InteractionLog, 0)
	for _, l := range m.logs[start:end] {
		out = append(out, *l)
	}
	return out, nil
}

func (m *mockInteractionRepo) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]domain.InteractionLog, error) {
	var out []domain.InteractionLog
	for _, l := range m.logs {
		if l.CustomerID == customerID {
			out = append(out, *l)
		}
	}
	return out, nil
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestGetOrCreateCustomer_NewCustomer(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	interactRepo := &mockInteractionRepo{}
	svc := synapse.NewService(customerRepo, interactRepo)

	profile, isNew, err := svc.GetOrCreateCustomer(context.Background(), domain.ChannelWhatsApp, "+966500000001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isNew {
		t.Error("expected isNew=true for first lookup")
	}
	if profile.ID == uuid.Nil {
		t.Error("profile ID should not be nil")
	}
	if profile.CustomerTier != domain.TierStandard {
		t.Errorf("expected standard tier, got %s", profile.CustomerTier)
	}
}

func TestGetOrCreateCustomer_ExistingCustomer(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	interactRepo := &mockInteractionRepo{}
	svc := synapse.NewService(customerRepo, interactRepo)

	p1, _, _ := svc.GetOrCreateCustomer(context.Background(), domain.ChannelWhatsApp, "+966500000001")
	p2, isNew, _ := svc.GetOrCreateCustomer(context.Background(), domain.ChannelWhatsApp, "+966500000001")

	if isNew {
		t.Error("expected isNew=false for second lookup")
	}
	if p1.ID != p2.ID {
		t.Error("same channel+externalID should resolve to the same profile")
	}
}

func TestGetOrCreateCustomer_DifferentChannels_DifferentProfiles(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	interactRepo := &mockInteractionRepo{}
	svc := synapse.NewService(customerRepo, interactRepo)

	p1, _, _ := svc.GetOrCreateCustomer(context.Background(), domain.ChannelWhatsApp, "+966500000001")
	p2, _, _ := svc.GetOrCreateCustomer(context.Background(), domain.ChannelMessenger, "fb_user_123")

	if p1.ID == p2.ID {
		t.Error("different channels should create separate profiles")
	}
}

func TestRecordTurn_PersistsLogAndTrace(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	interactRepo := &mockInteractionRepo{}
	svc := synapse.NewService(customerRepo, interactRepo)

	log := &domain.InteractionLog{
		CustomerID:  uuid.New(),
		Channel:     domain.ChannelWhatsApp,
		InboundMsg:  "Do you have widgets?",
		OutboundMsg: "Yes, we have Premium Widgets in stock!",
		TokensUsed:  150,
		LatencyMs:   1200,
	}

	trace := &domain.InteractionTrace{
		RetrievedContexts: []domain.RetrievedContext{
			{Content: "Our Premium Widget is handmade...", Score: 0.92},
		},
		SystemPrompt:     "You are Mera, a helpful retail assistant.",
		RawAgentThoughts: "The customer is asking about widgets. Let me check the catalog.",
		ToolsCalled: []domain.ToolCall{
			{ToolName: "knowledge_search", ArgsJSON: `{"query":"widgets"}`, ResultJSON: `{"chunks":[...]}`},
		},
	}

	err := svc.RecordTurn(context.Background(), log, trace)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// IDs should have been auto-generated.
	if log.ID == uuid.Nil {
		t.Error("log ID should be set after recording")
	}
	if trace.ID == uuid.Nil {
		t.Error("trace ID should be set after recording")
	}
	if trace.InteractionLogID != log.ID {
		t.Error("trace.InteractionLogID should match log.ID")
	}

	// Verify persistence.
	if len(interactRepo.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(interactRepo.logs))
	}
	if len(interactRepo.traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(interactRepo.traces))
	}
	if interactRepo.traces[0].SystemPrompt != "You are Mera, a helpful retail assistant." {
		t.Error("trace system prompt not persisted correctly")
	}
}

func TestUpdateSemanticSummary_UpdatesProfile(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	interactRepo := &mockInteractionRepo{}
	svc := synapse.NewService(customerRepo, interactRepo)

	// Create a customer first.
	profile, _, _ := svc.GetOrCreateCustomer(context.Background(), domain.ChannelWeb, "session_abc")

	err := svc.UpdateSemanticSummary(context.Background(), profile.ID, "Frequently asks about eco-friendly products.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify update via the mock.
	updated := customerRepo.profiles["web:session_abc"]
	if updated.SemanticSummary != "Frequently asks about eco-friendly products." {
		t.Errorf("summary not updated, got %q", updated.SemanticSummary)
	}
}

func TestListInteractions_Pagination(t *testing.T) {
	interactRepo := &mockInteractionRepo{}
	for i := 0; i < 10; i++ {
		interactRepo.logs = append(interactRepo.logs, &domain.InteractionLog{
			ID: uuid.New(), CustomerID: uuid.New(), InboundMsg: "msg", OutboundMsg: "reply",
		})
	}
	svc := synapse.NewService(newMockCustomerRepo(), interactRepo)

	page, err := svc.ListInteractions(context.Background(), 3, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page) != 3 {
		t.Errorf("expected 3 results, got %d", len(page))
	}

	page2, _ := svc.ListInteractions(context.Background(), 3, 8)
	if len(page2) != 2 {
		t.Errorf("expected 2 results for last page, got %d", len(page2))
	}
}
