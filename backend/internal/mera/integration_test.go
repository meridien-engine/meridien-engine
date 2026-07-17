package mera_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/mera"
	"github.com/meridien-engine/meridien-engine/internal/mera/agent"
	"github.com/meridien-engine/meridien-engine/internal/mera/hitl"
	"github.com/meridien-engine/meridien-engine/internal/mera/middleware"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	"github.com/shopspring/decimal"
)

// ─── Integration Mock Repositories ──────────────────────────────────────────

type mockIntegrationProductRepo struct {
	price decimal.Decimal
}

func (m *mockIntegrationProductRepo) GetBySKU(_ context.Context, sku string) (*domain.Product, error) {
	return &domain.Product{
		ID:         uuid.New(),
		BusinessID: uuid.New(),
		SKU:        sku,
		Name:       "Test Product",
		Price:      m.price,
		StockQty:   10,
		IsActive:   true,
	}, nil
}

func (m *mockIntegrationProductRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Product, error) {
	return nil, nil
}

func (m *mockIntegrationProductRepo) List(_ context.Context) ([]domain.Product, error) {
	return nil, nil
}

func (m *mockIntegrationProductRepo) DecrementStock(_ context.Context, id uuid.UUID, qty int32) (*domain.Product, error) {
	return nil, nil
}

func (m *mockIntegrationProductRepo) Create(_ context.Context, p *domain.Product) (*domain.Product, error) {
	return nil, nil
}

type mockIntegrationOrderRepo struct {
	mu           sync.Mutex
	ordersPlaced []*domain.Order
}

func (m *mockIntegrationOrderRepo) Create(_ context.Context, o *domain.Order) (*domain.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ordersPlaced = append(m.ordersPlaced, o)
	return o, nil
}

func (m *mockIntegrationOrderRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Order, error) {
	return nil, nil
}

func (m *mockIntegrationOrderRepo) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]domain.Order, error) {
	return nil, nil
}

func (m *mockIntegrationOrderRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	return nil, nil
}

func (m *mockIntegrationOrderRepo) ListOrders(ctx context.Context, limit, offset int32) ([]domain.Order, error) {
	return nil, nil
}

type mockIntegrationSecretsRepo struct{}

func (m *mockIntegrationSecretsRepo) UpsertSecret(ctx context.Context, businessID uuid.UUID, keyName string, plaintextVal string) (*domain.SystemSecret, error) {
	return nil, nil
}

func (m *mockIntegrationSecretsRepo) GetSecret(ctx context.Context, businessID uuid.UUID, keyName string) (string, error) {
	return "", nil
}

func (m *mockIntegrationSecretsRepo) ListSecretKeys(ctx context.Context, businessID uuid.UUID) ([]domain.SystemSecret, error) {
	return nil, nil
}

func (m *mockIntegrationSecretsRepo) DeleteSecret(ctx context.Context, businessID uuid.UUID, keyName string) error {
	return nil
}

type mockIntegrationCustomerRepo struct {
	profiles map[string]*domain.CustomerProfile
}

func (m *mockIntegrationCustomerRepo) GetOrCreateByChannel(
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

func (m *mockIntegrationCustomerRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	return nil, nil
}

func (m *mockIntegrationCustomerRepo) UpdateSemanticSummary(_ context.Context, id uuid.UUID, summary string) error {
	return nil
}

func (m *mockIntegrationCustomerRepo) UpdateTier(_ context.Context, id uuid.UUID, tier domain.CustomerTier) error {
	return nil
}

type mockIntegrationInteractionRepo struct {
	mu           sync.Mutex
	logs         []*domain.InteractionLog
	traces       []*domain.InteractionTrace
	timedOutRuns map[uuid.UUID]bool
}

func (m *mockIntegrationInteractionRepo) RecordTurn(_ context.Context, log *domain.InteractionLog, trace *domain.InteractionTrace) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, log)
	m.traces = append(m.traces, trace)
	return nil
}

func (m *mockIntegrationInteractionRepo) GetWithTrace(_ context.Context, id uuid.UUID) (*domain.InteractionLog, *domain.InteractionTrace, error) {
	return nil, nil, nil
}

func (m *mockIntegrationInteractionRepo) List(_ context.Context, limit, offset int32) ([]domain.InteractionLog, error) {
	return nil, nil
}

func (m *mockIntegrationInteractionRepo) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]domain.InteractionLog, error) {
	return nil, nil
}

func (m *mockIntegrationInteractionRepo) GetExpiredSuspensions(ctx context.Context) ([]hitl.ExpiredSuspension, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []hitl.ExpiredSuspension
	for _, t := range m.traces {
		if t.HITLStatus == domain.HITLStatusPending {
			out = append(out, hitl.ExpiredSuspension{
				TraceID:    t.ID,
				WorkflowID: t.WorkflowID,
				ExpiresAt:  time.Now().Add(-1 * time.Hour), // pretend it expired
			})
		}
	}
	return out, nil
}

func (m *mockIntegrationInteractionRepo) MarkTimedOut(ctx context.Context, traceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timedOutRuns[traceID] = true
	// Update mock trace status in the slice
	for _, t := range m.traces {
		if t.ID == traceID {
			t.HITLStatus = domain.HITLStatusTimedOut
		}
	}
	return nil
}

// Helper to construct a base64-encoded mock JWT string without signature verification
func makeMockTokenForIntegration(businessID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"business_id":"` + businessID + `"}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return header + "." + payload + "." + sig
}

// TestMeraWorkflow_PriceMismatch_HITLSuspension_And_Timeout checks that:
// 1. A client webhook request with a pricing discrepancy suspends the workflow graph.
// 2. An interaction trace is successfully written to the DB with status 'pending' (HITL suspension).
// 3. The background HITL checker polls the DB, finds the expired suspension, and transitions it to 'timed_out'.
func TestMeraWorkflow_PriceMismatch_HITLSuspension_And_Timeout(t *testing.T) {
	custRepo := &mockIntegrationCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
	intRepo := &mockIntegrationInteractionRepo{
		timedOutRuns: make(map[uuid.UUID]bool),
	}
	synSvc := synapse.NewService(custRepo, intRepo)

	// Catalog price of WIDGET-01 is $100
	pRepo := &mockIntegrationProductRepo{price: decimal.NewFromInt(100)}
	oRepo := &mockIntegrationOrderRepo{}
	erpSvc := erp.NewService(pRepo, oRepo)

	kRepo := &mockKnowledgeRepo{}

	h, err := mera.NewHandler(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo, &mockIntegrationSecretsRepo{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	// 1. Dispatch Webhook message simulating checkout with price mismatch
	// Customer expects $10, catalog total is $100.
	reqBody := mera.WebhookRequest{
		Channel:           "whatsapp",
		ChannelExternalID: "+1234567890",
		Message:           "I want to order widget-01",
		ExpectedPrice:     10.0,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mera/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	bizID := uuid.New().String()
	req.Header.Set("Authorization", "Bearer "+makeMockTokenForIntegration(bizID))

	rr := httptest.NewRecorder()
	middleware.JWTAuth(http.HandlerFunc(h.Webhook)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp mera.WebhookResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 2. Verify that the workflow was suspended and returned the approval prompt
	if !strings.Contains(resp.Reply, "Price mismatch") {
		t.Errorf("expected reply to contain mismatch warning, got %q", resp.Reply)
	}

	// 3. Verify that the order WAS NOT placed (held in suspension)
	if len(oRepo.ordersPlaced) != 0 {
		t.Errorf("expected no order to be placed yet, got %d orders", len(oRepo.ordersPlaced))
	}

	// 4. Verify that the interaction trace was written to the DB with status 'pending'
	intRepo.mu.Lock()
	if len(intRepo.traces) != 1 {
		intRepo.mu.Unlock()
		t.Fatalf("expected 1 interaction trace recorded, got %d", len(intRepo.traces))
	}
	trace := intRepo.traces[0]
	intRepo.mu.Unlock()

	if trace.HITLStatus != domain.HITLStatusPending {
		t.Errorf("expected HITL status to be pending, got %s", trace.HITLStatus)
	}
	if trace.WorkflowID == "" {
		t.Error("expected non-empty WorkflowID for suspended run")
	}

	// 5. Run the background HITL Checker loop once to simulate expiry processing
	checker := hitl.New(intRepo, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Running a single check runOnce
	// We'll call checker's private runOnce or run a ticker tick
	go checker.Run(ctx)
	time.Sleep(30 * time.Millisecond)
	cancel()

	// 6. Verify that the checker updated the trace to 'timed_out'
	intRepo.mu.Lock()
	defer intRepo.mu.Unlock()
	if !intRepo.timedOutRuns[trace.ID] {
		t.Errorf("expected trace ID %s to be marked timed out", trace.ID)
	}
	if trace.HITLStatus != domain.HITLStatusTimedOut {
		t.Errorf("expected trace status to transition to timed_out, got %s", trace.HITLStatus)
	}
}
