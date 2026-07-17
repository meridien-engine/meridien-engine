package mera_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/mera"
	"github.com/meridien-engine/meridien-engine/internal/mera/agent"
	"github.com/meridien-engine/meridien-engine/internal/mera/middleware"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	"github.com/shopspring/decimal"
)

// ─── Mock Repositories ────────────────────────────────────────────────────────

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

func (m *mockOrderRepo) ListOrders(ctx context.Context, limit, offset int32) ([]domain.Order, error) {
	return nil, nil
}

type mockSecretsRepo struct{}

func (m *mockSecretsRepo) UpsertSecret(ctx context.Context, businessID uuid.UUID, keyName string, plaintextVal string) (*domain.SystemSecret, error) {
	return nil, nil
}

func (m *mockSecretsRepo) GetSecret(ctx context.Context, businessID uuid.UUID, keyName string) (string, error) {
	return "", nil
}

func (m *mockSecretsRepo) ListSecretKeys(ctx context.Context, businessID uuid.UUID) ([]domain.SystemSecret, error) {
	return nil, nil
}

func (m *mockSecretsRepo) DeleteSecret(ctx context.Context, businessID uuid.UUID, keyName string) error {
	return nil
}

type mockCustomerRepo struct {
	profiles map[string]*domain.CustomerProfile
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
	return nil, nil
}

func (m *mockCustomerRepo) UpdateSemanticSummary(_ context.Context, id uuid.UUID, summary string) error {
	return nil
}

func (m *mockCustomerRepo) UpdateTier(_ context.Context, id uuid.UUID, tier domain.CustomerTier) error {
	return nil
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

// Helper to construct a base64-encoded mock JWT string without signature verification
func makeMockToken(businessID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"business_id":"` + businessID + `"}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return header + "." + payload + "." + sig
}

func TestWebhookHandler_Success(t *testing.T) {
	custRepo := &mockCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
	intRepo := &mockInteractionRepo{}
	synSvc := synapse.NewService(custRepo, intRepo)

	pRepo := &mockProductRepo{}
	oRepo := &mockOrderRepo{}
	erpSvc := erp.NewService(pRepo, oRepo)

	kRepo := &mockKnowledgeRepo{}

	h, err := mera.NewHandler(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo, &mockSecretsRepo{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	// Create test request body
	reqBody := mera.WebhookRequest{
		Channel:           "whatsapp",
		ChannelExternalID: "+1234567890",
		Message:           "What is the catalog price?",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mera/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Set valid auth header
	bizID := uuid.New().String()
	req.Header.Set("Authorization", "Bearer "+makeMockToken(bizID))

	rr := httptest.NewRecorder()

	// Wrap handler in auth middleware
	middleware.JWTAuth(http.HandlerFunc(h.Webhook)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp mera.WebhookResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Reply, "Mock Gemini response") {
		t.Errorf("expected reply to contain workflow text, got %q", resp.Reply)
	}

	if len(intRepo.logs) != 1 {
		t.Errorf("expected 1 log recorded, got %d", len(intRepo.logs))
	}
}

func TestWebhookHandler_Unauthorized(t *testing.T) {
	custRepo := &mockCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
	intRepo := &mockInteractionRepo{}
	synSvc := synapse.NewService(custRepo, intRepo)

	pRepo := &mockProductRepo{}
	oRepo := &mockOrderRepo{}
	erpSvc := erp.NewService(pRepo, oRepo)

	kRepo := &mockKnowledgeRepo{}

	h, err := mera.NewHandler(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo, &mockSecretsRepo{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	reqBody := mera.WebhookRequest{
		Channel:           "whatsapp",
		ChannelExternalID: "+1234567890",
		Message:           "What is the catalog price?",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mera/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	rr := httptest.NewRecorder()
	middleware.JWTAuth(http.HandlerFunc(h.Webhook)).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestWebhookHandler_BadRequest(t *testing.T) {
	custRepo := &mockCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
	intRepo := &mockInteractionRepo{}
	synSvc := synapse.NewService(custRepo, intRepo)

	pRepo := &mockProductRepo{}
	oRepo := &mockOrderRepo{}
	erpSvc := erp.NewService(pRepo, oRepo)

	kRepo := &mockKnowledgeRepo{}

	h, err := mera.NewHandler(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo, &mockSecretsRepo{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	// Missing fields
	reqBody := mera.WebhookRequest{
		Channel: "whatsapp",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mera/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	bizID := uuid.New().String()
	req.Header.Set("Authorization", "Bearer "+makeMockToken(bizID))

	rr := httptest.NewRecorder()
	middleware.JWTAuth(http.HandlerFunc(h.Webhook)).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestWebhookHandler_MetaVerification(t *testing.T) {
	custRepo := &mockCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
	intRepo := &mockInteractionRepo{}
	synSvc := synapse.NewService(custRepo, intRepo)
	pRepo := &mockProductRepo{}
	oRepo := &mockOrderRepo{}
	erpSvc := erp.NewService(pRepo, oRepo)
	kRepo := &mockKnowledgeRepo{}

	h, err := mera.NewHandler(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo, &mockSecretsRepo{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	t.Setenv("META_VERIFY_TOKEN", "custom_secret_token")

	// 1. Success verification handshake
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mera/webhook?hub.mode=subscribe&hub.verify_token=custom_secret_token&hub.challenge=test_challenge_123", nil)
	rr := httptest.NewRecorder()
	h.Webhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "test_challenge_123" {
		t.Errorf("expected challenge string back, got %q", rr.Body.String())
	}

	// 2. Forbidden verification (invalid token)
	reqForbidden := httptest.NewRequest(http.MethodGet, "/api/v1/mera/webhook?hub.mode=subscribe&hub.verify_token=wrong_token&hub.challenge=test_challenge_123", nil)
	rrForbidden := httptest.NewRecorder()
	h.Webhook(rrForbidden, reqForbidden)

	if rrForbidden.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rrForbidden.Code)
	}
}

func TestWebhookHandler_MetaMessengerPayload(t *testing.T) {
	custRepo := &mockCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
	intRepo := &mockInteractionRepo{}
	synSvc := synapse.NewService(custRepo, intRepo)
	pRepo := &mockProductRepo{}
	oRepo := &mockOrderRepo{}
	erpSvc := erp.NewService(pRepo, oRepo)
	kRepo := &mockKnowledgeRepo{}

	h, err := mera.NewHandler(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo, &mockSecretsRepo{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	// Messenger nested payload
	payload := `{
		"object": "page",
		"entry": [{
			"id": "page-123",
			"messaging": [{
				"sender": { "id": "messenger-user-99" },
				"recipient": { "id": "page-123" },
				"timestamp": 12345678,
				"message": { "mid": "mid.1", "text": "hello from messenger" }
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mera/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	bizID := uuid.New().String()
	req.Header.Set("Authorization", "Bearer "+makeMockToken(bizID))

	rr := httptest.NewRecorder()
	middleware.JWTAuth(http.HandlerFunc(h.Webhook)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp mera.WebhookResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Reply, "Mock Gemini response") {
		t.Errorf("expected reply to contain workflow text, got %q", resp.Reply)
	}
}

func TestWebhookHandler_MetaWhatsAppPayload(t *testing.T) {
	custRepo := &mockCustomerRepo{profiles: make(map[string]*domain.CustomerProfile)}
	intRepo := &mockInteractionRepo{}
	synSvc := synapse.NewService(custRepo, intRepo)
	pRepo := &mockProductRepo{}
	oRepo := &mockOrderRepo{}
	erpSvc := erp.NewService(pRepo, oRepo)
	kRepo := &mockKnowledgeRepo{}

	h, err := mera.NewHandler(&agent.MockLLM{}, synSvc, erpSvc, pRepo, kRepo, &mockSecretsRepo{})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	// WhatsApp Cloud API nested payload
	payload := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "wa-acc-123",
			"changes": [{
				"value": {
					"messaging_product": "whatsapp",
					"metadata": { "display_phone_number": "16505551111", "phone_number_id": "phone-123" },
					"contacts": [{ "profile": { "name": "Test User" }, "wa_id": "16505552222" }],
					"messages": [{
						"from": "+16505552222",
						"id": "msg-123",
						"timestamp": "12345678",
						"type": "text",
						"text": { "body": "hello from whatsapp" }
					}]
				},
				"field": "messages"
			}]
		}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mera/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	bizID := uuid.New().String()
	req.Header.Set("Authorization", "Bearer "+makeMockToken(bizID))

	rr := httptest.NewRecorder()
	middleware.JWTAuth(http.HandlerFunc(h.Webhook)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp mera.WebhookResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(resp.Reply, "Mock Gemini response") {
		t.Errorf("expected reply to contain workflow text, got %q", resp.Reply)
	}
}

