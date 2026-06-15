// Package grpchandler — Synapse gRPC handler.
package grpchandler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SynapseHandler implements the gRPC SynapseService server interface.
type SynapseHandler struct {
	svc *synapse.Service
}

func NewSynapseHandler(svc *synapse.Service) *SynapseHandler {
	return &SynapseHandler{svc: svc}
}

// GetCustomerProfile resolves the UCM from a channel identifier.
// Called by Mera at the start of every conversation turn.
func (h *SynapseHandler) GetCustomerProfile(ctx context.Context, req *GetCustomerProfileRequest) (*GetCustomerProfileResponse, error) {
	if req.ChannelType == "" || req.ChannelExternalID == "" {
		return nil, status.Error(codes.InvalidArgument, "channel_type and channel_external_id are required")
	}

	profile, isNew, err := h.svc.GetOrCreateCustomer(ctx,
		domain.ChannelType(req.ChannelType),
		req.ChannelExternalID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to resolve customer profile")
	}

	return &GetCustomerProfileResponse{
		CustomerID:      profile.ID.String(),
		UnifiedName:     profile.UnifiedName,
		CustomerTier:    string(profile.CustomerTier),
		SemanticSummary: profile.SemanticSummary,
		IsNewCustomer:   isNew,
	}, nil
}

// RecordInteraction persists the full interaction log + observability trace.
// Called by Mera synchronously after producing its response.
func (h *SynapseHandler) RecordInteraction(ctx context.Context, req *RecordInteractionRequest) (*RecordInteractionResponse, error) {
	if req.CustomerID == "" || req.InboundMsg == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id and inbound_msg are required")
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id")
	}

	// Map retrieved contexts.
	contexts := make([]domain.RetrievedContext, len(req.RetrievedContexts))
	for i, c := range req.RetrievedContexts {
		contexts[i] = domain.RetrievedContext{Content: c.Content, Score: c.Score}
	}

	// Map tool calls.
	tools := make([]domain.ToolCall, len(req.ToolsCalled))
	for i, t := range req.ToolsCalled {
		tools[i] = domain.ToolCall{
			ToolName:   t.ToolName,
			ArgsJSON:   t.ArgsJSON,
			ResultJSON: t.ResultJSON,
		}
	}

	log := &domain.InteractionLog{
		CustomerID:  customerID,
		Channel:     domain.ChannelType(req.Channel),
		InboundMsg:  req.InboundMsg,
		OutboundMsg: req.OutboundMsg,
		TokensUsed:  req.TokensUsed,
		LatencyMs:   req.LatencyMs,
	}

	trace := &domain.InteractionTrace{
		RetrievedContexts: contexts,
		SystemPrompt:      req.SystemPrompt,
		RawAgentThoughts:  req.RawAgentThoughts,
		ToolsCalled:       tools,
	}

	if err := h.svc.RecordTurn(ctx, log, trace); err != nil {
		return nil, status.Error(codes.Internal, "failed to record interaction")
	}

	// Update Prometheus agent metrics.
	metrics.AgentTurnsTotal.WithLabelValues(req.Channel, "success").Inc()
	metrics.AgentLatency.WithLabelValues(req.Channel).Observe(
		time.Duration(req.LatencyMs).Seconds() / 1000,
	)
	metrics.LLMTokensUsed.WithLabelValues("total").Add(float64(req.TokensUsed))

	return &RecordInteractionResponse{
		InteractionID: log.ID.String(),
		Success:       true,
	}, nil
}

// ─── DTO types ────────────────────────────────────────────────────────────────

type GetCustomerProfileRequest struct {
	ChannelType       string
	ChannelExternalID string
}

type GetCustomerProfileResponse struct {
	CustomerID      string
	UnifiedName     string
	CustomerTier    string
	SemanticSummary string
	IsNewCustomer   bool
}

type RetrievedContextDTO struct {
	Content string
	Score   float64
}

type ToolCallDTO struct {
	ToolName   string
	ArgsJSON   string
	ResultJSON string
}

type RecordInteractionRequest struct {
	CustomerID        string
	Channel           string
	InboundMsg        string
	OutboundMsg       string
	TokensUsed        int32
	LatencyMs         int32
	SystemPrompt      string
	RawAgentThoughts  string
	RetrievedContexts []*RetrievedContextDTO
	ToolsCalled       []*ToolCallDTO
}

type RecordInteractionResponse struct {
	InteractionID string
	Success       bool
}
