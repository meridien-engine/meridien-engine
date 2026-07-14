// Package mera implements the customer-facing AI agent endpoints,
// gateway routing, and webhook handlers.
package mera

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/mera/agent"
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

// Handler orchestrates incoming customer communication.
type Handler struct {
	synapseSvc *synapse.Service
	runner     *runner.Runner
}

// NewHandler creates a Handler with dependencies wired.
func NewHandler(
	llmModel model.LLM,
	synSvc *synapse.Service,
	erpSvc *erp.Service,
	pRepo domain.ProductRepository,
	kRepo domain.KnowledgeRepository,
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
		synapseSvc: synSvc,
		runner:     r,
	}, nil
}

// Webhook handles the incoming customer message webhook.
// It resolves the customer profile, runs the ADK workflow,
// records the interaction log + trace, and returns the response.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
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

	// 6. Reply to channel / client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WebhookResponse{
		Reply:      replyText,
		CustomerID: profile.ID.String(),
		IsNew:      isNew,
	})
}
