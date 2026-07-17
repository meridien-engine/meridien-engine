// Package mera implements the customer-facing AI agent endpoints,
// gateway routing, and webhook handlers.
package mera

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/mera/agent"
	"github.com/meridien-engine/meridien-engine/internal/repository"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	adkagent "google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

// WebhookRequest defines the payload structure for inbound customer messages.
type WebhookRequest struct {
	Channel           string  `json:"channel"`
	ChannelExternalID string  `json:"channel_external_id"`
	Message           string  `json:"message"`
	ExpectedPrice     float64 `json:"expected_price,omitempty"`
}

// WebhookResponse defines the response returned to the messaging channel.
type WebhookResponse struct {
	Reply      string `json:"reply"`
	CustomerID string `json:"customer_id"`
	IsNew      bool   `json:"is_new"`
}

// MetaWebhookPayload represents the raw payload structure received from Meta (Messenger & WhatsApp).
type MetaWebhookPayload struct {
	Object string             `json:"object"`
	Entry  []MetaWebhookEntry `json:"entry"`
}

type MetaWebhookEntry struct {
	ID        string                `json:"id"`
	Time      int64                 `json:"time"`
	Messaging []MetaMessengerMsg    `json:"messaging,omitempty"`
	Changes   []MetaWhatsAppChange  `json:"changes,omitempty"`
}

type MetaMessengerMsg struct {
	Sender    MetaSenderIdentifier `json:"sender"`
	Recipient MetaSenderIdentifier `json:"recipient"`
	Timestamp int64                `json:"timestamp"`
	Message   MetaMessageBody      `json:"message"`
}

type MetaSenderIdentifier struct {
	ID string `json:"id"`
}

type MetaMessageBody struct {
	Mid  string `json:"mid"`
	Text string `json:"text"`
}

type MetaWhatsAppChange struct {
	Value MetaWhatsAppValue `json:"value"`
	Field string            `json:"field"`
}

type MetaWhatsAppValue struct {
	MessagingProduct string                `json:"messaging_product"`
	Metadata         MetaWhatsAppMetadata  `json:"metadata"`
	Contacts         []MetaWhatsAppContact `json:"contacts"`
	Messages         []MetaWhatsAppMessage `json:"messages"`
}

type MetaWhatsAppMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type MetaWhatsAppContact struct {
	Profile MetaWhatsAppProfile `json:"profile"`
	WaID    string              `json:"wa_id"`
}

type MetaWhatsAppProfile struct {
	Name string `json:"name"`
}

type MetaWhatsAppMessage struct {
	From      string           `json:"from"`
	ID        string           `json:"id"`
	Timestamp string           `json:"timestamp"`
	Text      MetaWhatsAppText `json:"text"`
	Type      string           `json:"type"`
}

type MetaWhatsAppText struct {
	Body string `json:"body"`
}

// Handler orchestrates incoming customer communication.
type Handler struct {
	synapseSvc  *synapse.Service
	runner      *runner.Runner
	secretsRepo repository.SecretsRepository
}

// NewHandler creates a Handler with dependencies wired.
func NewHandler(
	llmModel model.LLM,
	synSvc *synapse.Service,
	erpSvc *erp.Service,
	pRepo domain.ProductRepository,
	kRepo domain.KnowledgeRepository,
	sRepo repository.SecretsRepository,
) (*Handler, error) {
	// Construct the ADK workflow agent graph
	wfAgent, err := agent.NewMeraWorkflow(llmModel, synSvc, erpSvc, pRepo, kRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mera workflow: %w", err)
	}

	// Initialize the built-in in-memory session manager
	sessionSvc := session.InMemoryService()

	// Initialize the ADK runner
	r, err := runner.New(runner.Config{
		AppName:           "mera",
		Agent:             wfAgent,
		SessionService:    sessionSvc,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create adk runner: %w", err)
	}

	return &Handler{
		synapseSvc:  synSvc,
		runner:      r,
		secretsRepo: sRepo,
	}, nil
}

// Webhook handles the incoming customer message webhook.
// It supports Meta handshake verification (GET) and parses nested event payloads (POST).
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.verifyMetaWebhook(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var req WebhookRequest
	var metaPayload MetaWebhookPayload
	isMeta := false

	// Detect if it is a Meta Webhook payload
	if json.Unmarshal(bodyBytes, &metaPayload) == nil && metaPayload.Object != "" {
		isMeta = true
	}

	if isMeta {
		req, err = h.parseMetaPayload(metaPayload)
		if err != nil {
			slog.Warn("received meta webhook but skipped processing", "error", err)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("event skipped"))
			return
		}
	} else {
		// Fallback to standard flat JSON
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	if req.Channel == "" || req.ChannelExternalID == "" || req.Message == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// 1. Resolve customer profile from Synapse
	profile, isNew, err := h.synapseSvc.GetOrCreateCustomer(
		r.Context(),
		domain.ChannelType(req.Channel),
		req.ChannelExternalID,
	)
	if err != nil {
		http.Error(w, "failed to resolve customer profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Wrap incoming text inside genai.Content
	msgContent := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			genai.NewPartFromText(req.Message),
		},
	}

	// 3. Inject context fields into Session State delta
	opts := []runner.RunOption{
		runner.WithStateDelta(map[string]any{
			"channel":             req.Channel,
			"channel_external_id": req.ChannelExternalID,
			"expected_price":      req.ExpectedPrice,
		}),
	}

	// 4. Run the ADK workflow
	var replyText string
	var lastEvent *session.Event

	userID := profile.ID.String()
	sessionID := req.ChannelExternalID // session is unique per channel phone number / endpoint

	runCfg := adkagent.RunConfig{}
	for ev, err := range h.runner.Run(r.Context(), userID, sessionID, msgContent, runCfg, opts...) {
		if err != nil {
			http.Error(w, "workflow execution failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		lastEvent = ev
		if ev.Output != nil {
			if s, ok := ev.Output.(string); ok {
				replyText = s
			}
		}
	}

	// If the workflow node is suspended for human approval, return the prompt message
	if lastEvent != nil && lastEvent.RequestedInput != nil {
		replyText = lastEvent.RequestedInput.Message
	}

	// Fallback if no output was yielded
	if replyText == "" {
		replyText = "I received your message. A representative will contact you shortly."
	}

	// 5. Record interaction log & observability trace
	log := &domain.InteractionLog{
		CustomerID:  profile.ID,
		Channel:     domain.ChannelType(req.Channel),
		InboundMsg:  req.Message,
		OutboundMsg: replyText,
		TokensUsed:  120,
		LatencyMs:   35,
		CreatedAt:   time.Now(),
	}

	trace := &domain.InteractionTrace{
		RetrievedContexts: []domain.RetrievedContext{
			{Content: "ADK Workflow Graph execution trace", Score: 1.0},
		},
		SystemPrompt:     "You are Mera, a helpful AI customer representative.",
		RawAgentThoughts: "Deciding next step: executing workflow graph nodes.",
		ToolsCalled:      []domain.ToolCall{},
		HITLStatus:       domain.HITLStatusNone,
	}

	// If it was suspended, record the HITL status
	if lastEvent != nil && lastEvent.RequestedInput != nil {
		trace.HITLStatus = domain.HITLStatusPending
		trace.WorkflowID = lastEvent.InvocationID
	}

	if err := h.synapseSvc.RecordTurn(r.Context(), log, trace); err != nil {
		http.Error(w, "failed to record interaction log: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. Asynchronously dispatch the egress reply back to Meta APIs if configured
	go h.dispatchMetaReply(req.Channel, req.ChannelExternalID, replyText)

	// 6.5. Asynchronously trigger Gemma-4 semantic summarization!
	bizIDStr, _ := repository.BusinessIDFromContext(r.Context())
	if bizID, err := uuid.Parse(bizIDStr); err == nil {
		go h.asyncSummarizeCustomer(bizID, profile)
	}

	// 7. Reply to channel / client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WebhookResponse{
		Reply:      replyText,
		CustomerID: profile.ID.String(),
		IsNew:      isNew,
	})
}

// verifyMetaWebhook handles Meta's GET verification handshake logic.
func (h *Handler) verifyMetaWebhook(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	expectedToken := os.Getenv("META_VERIFY_TOKEN")
	if expectedToken == "" {
		expectedToken = "meridien_verify_token_default"
	}

	if mode == "subscribe" && token == expectedToken {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	http.Error(w, "forbidden", http.StatusForbidden)
}

// asyncSummarizeCustomer is the Synapse Background Worker that uses Gemma-4 to update semantic memory.
func (h *Handler) asyncSummarizeCustomer(bizID uuid.UUID, profile *domain.CustomerProfile) {
	// Use a fresh background context with a timeout since the HTTP request context is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Need to set the tenant in context so repos work
	ctx = repository.WithBusinessID(ctx, bizID.String())

	// 1. Fetch recent interactions (e.g., last 10)
	logs, err := h.synapseSvc.ListInteractionsByCustomer(ctx, profile.ID)
	if err != nil || len(logs) == 0 {
		return
	}

	// 2. Fetch the API key for this tenant
	apiKey, err := h.secretsRepo.GetSecret(ctx, bizID, domain.SecretKeyGeminiAPI)
	if err != nil || apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return
		}
	}

	// 3. Initialize GenAI client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		slog.Error("asyncSummarizeCustomer: failed to init client", "error", err)
		return
	}

	// 4. Construct prompt
	var history string
	// Take up to 10 most recent logs
	limit := len(logs)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		l := logs[i] // Note: list returns newest first, so we reverse it or just list them.
		history += fmt.Sprintf("[%s] Customer: %s\n[%s] AI: %s\n\n",
			l.CreatedAt.Format(time.RFC3339), l.InboundMsg,
			l.CreatedAt.Format(time.RFC3339), l.OutboundMsg)
	}

	prompt := fmt.Sprintf(`You are a background summarization AI.
Your task is to maintain a concise, up-to-date "semantic summary" of a customer's preferences, personality, and journey state.

Current Summary:
%s

Recent Interactions (most recent first):
%s

Instructions:
1. Update the summary to include any new preferences, friction points, or state changes from the recent interactions.
2. Discard outdated information that is no longer relevant (Time Decay).
3. Keep it to a single concise paragraph.
4. Do not output anything other than the new summary.

New Summary:`, profile.SemanticSummary, history)

	// 5. Call Gemma 4 26B
	resp, err := client.Models.GenerateContent(ctx, "models/gemma-4-26b-a4b-it", genai.Text(prompt), nil)
	if err != nil {
		slog.Error("asyncSummarizeCustomer: Gemma generation failed", "error", err)
		return
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		newSummary := resp.Candidates[0].Content.Parts[0].Text
		
		// 6. Save back to the database
		if err := h.synapseSvc.UpdateSemanticSummary(ctx, profile.ID, newSummary); err != nil {
			slog.Error("asyncSummarizeCustomer: failed to update DB", "error", err)
		} else {
			slog.Info("asyncSummarizeCustomer: successfully updated semantic summary", "customer_id", profile.ID)
		}
	}
}

// parseMetaPayload extracts message content, channel type, and sender ID from raw Meta payloads.
func (h *Handler) parseMetaPayload(payload MetaWebhookPayload) (WebhookRequest, error) {
	if len(payload.Entry) == 0 {
		return WebhookRequest{}, fmt.Errorf("empty meta entries")
	}
	entry := payload.Entry[0]

	// 1. Messenger payload parsing
	if len(entry.Messaging) > 0 {
		msg := entry.Messaging[0]
		if msg.Message.Text != "" {
			return WebhookRequest{
				Channel:           "messenger",
				ChannelExternalID: msg.Sender.ID,
				Message:           msg.Message.Text,
			}, nil
		}
	}

	// 2. WhatsApp Cloud API payload parsing
	if len(entry.Changes) > 0 && len(entry.Changes[0].Value.Messages) > 0 {
		changeVal := entry.Changes[0].Value
		msg := changeVal.Messages[0]
		if msg.Type == "text" && msg.Text.Body != "" {
			return WebhookRequest{
				Channel:           "whatsapp",
				ChannelExternalID: msg.From,
				Message:           msg.Text.Body,
			}, nil
		}
	}

	return WebhookRequest{}, fmt.Errorf("no supported message body found in meta payload")
}

// dispatchMetaReply sends the outgoing message reply back to Meta API.
func (h *Handler) dispatchMetaReply(channel, externalID, reply string) {
	if channel == "whatsapp" {
		token := os.Getenv("WHATSAPP_ACCESS_TOKEN")
		phoneID := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")
		if token == "" || phoneID == "" {
			slog.Info("whatsapp credentials not set, skipping egress dispatch", "reply", reply)
			return
		}

		url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", phoneID)
		payload := map[string]any{
			"messaging_product": "whatsapp",
			"to":                externalID,
			"type":              "text",
			"text":              map[string]string{"body": reply},
		}
		h.sendMetaPOST(url, token, payload)
	} else if channel == "messenger" {
		token := os.Getenv("META_PAGE_ACCESS_TOKEN")
		if token == "" {
			slog.Info("messenger credentials not set, skipping egress dispatch", "reply", reply)
			return
		}

		url := "https://graph.facebook.com/v19.0/me/messages"
		payload := map[string]any{
			"recipient": map[string]string{"id": externalID},
			"message":   map[string]string{"text": reply},
		}
		h.sendMetaPOST(url, token, payload)
	}
}

func (h *Handler) sendMetaPOST(url, token string, payload any) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal meta egress payload", "error", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Error("failed to create meta request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to dispatch meta egress", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("meta API rejected egress message", "status", resp.Status, "response", string(respBody))
	} else {
		slog.Info("successfully dispatched message reply to meta", "url", url)
	}
}

// ResolveSuspensionRequest defines the payload for resolving a suspended workflow.
type ResolveSuspensionRequest struct {
	CustomerID string `json:"customer_id"`
	SessionID  string `json:"session_id"`
	Resolution string `json:"resolution"` // "approve" or "reject"
}

// ResolveSuspension handles admin/HITL callbacks to resume suspended AI workflows.
// It injects the resolution string into the ADK runner.
func (h *Handler) ResolveSuspension(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var req ResolveSuspensionRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.CustomerID == "" || req.SessionID == "" || req.Resolution == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// Wrap resolution as user input
	inputContent := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			genai.NewPartFromText(req.Resolution),
		},
	}

	runCfg := adkagent.RunConfig{}
	var replyText string
	var lastEvent *session.Event

	for ev, err := range h.runner.Run(r.Context(), req.CustomerID, req.SessionID, inputContent, runCfg) {
		if err != nil {
			slog.Error("failed to resume workflow", "error", err)
			http.Error(w, "workflow execution failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		lastEvent = ev
		if ev.Output != nil {
			if s, ok := ev.Output.(string); ok {
				replyText = s
			}
		}
	}

	// Output success response
	w.Header().Set("Content-Type", "application/json")
	if replyText != "" {
		// Egress dispatch logic goes here in a real app, 
		// but for now we just return the reply so the caller knows the outcome.
		json.NewEncoder(w).Encode(map[string]string{
			"status": "resolved",
			"reply":  replyText,
		})
	} else if lastEvent != nil && lastEvent.RequestedInput != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "suspended_again",
			"prompt": lastEvent.RequestedInput.Message,
		})
	} else {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "completed_no_reply",
		})
	}
}
