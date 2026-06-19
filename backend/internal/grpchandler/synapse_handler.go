// Package grpchandler — Synapse gRPC handler.
package grpchandler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	pb "github.com/meridien-engine/meridien-engine/internal/gen/synapse"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SynapseHandler implements the gRPC SynapseService server interface.
type SynapseHandler struct {
	pb.UnimplementedSynapseServiceServer
	svc *synapse.Service
}

func NewSynapseHandler(svc *synapse.Service) *SynapseHandler {
	return &SynapseHandler{svc: svc}
}

// GetCustomerProfile resolves the UCM from a channel identifier.
// Called by Mera at the start of every conversation turn.
func (h *SynapseHandler) GetCustomerProfile(ctx context.Context, req *pb.ProfileRequest) (*pb.ProfileResponse, error) {
	if req.ChannelType == "" || req.ChannelExternalId == "" {
		return nil, status.Error(codes.InvalidArgument, "channel_type and channel_external_id are required")
	}

	profile, isNew, err := h.svc.GetOrCreateCustomer(ctx,
		domain.ChannelType(req.ChannelType),
		req.ChannelExternalId,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to resolve customer profile")
	}

	return &pb.ProfileResponse{
		CustomerId:      profile.ID.String(),
		UnifiedName:     profile.UnifiedName,
		CustomerTier:    string(profile.CustomerTier),
		SemanticSummary: profile.SemanticSummary,
		IsNewCustomer:   isNew,
	}, nil
}

// RecordInteraction persists the full interaction log + observability trace.
// Called by Mera synchronously after producing its response.
func (h *SynapseHandler) RecordInteraction(ctx context.Context, req *pb.InteractionRecord) (*pb.InteractionResponse, error) {
	if req.CustomerId == "" || req.InboundMsg == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id and inbound_msg are required")
	}

	customerID, err := uuid.Parse(req.CustomerId)
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
			ArgsJSON:   t.ArgsJson,
			ResultJSON: t.ResultJson,
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

	return &pb.InteractionResponse{
		InteractionId: log.ID.String(),
		Success:       true,
	}, nil
}

// GetCustomerByID retrieves a customer profile by ID.
func (h *SynapseHandler) GetCustomerByID(ctx context.Context, req *pb.GetCustomerByIDRequest) (*pb.GetCustomerByIDResponse, error) {
	id, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id UUID")
	}
	profile, err := h.svc.GetCustomerByID(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get customer profile: "+err.Error())
	}
	return &pb.GetCustomerByIDResponse{
		CustomerId:      profile.ID.String(),
		UnifiedName:     profile.UnifiedName,
		CustomerTier:    string(profile.CustomerTier),
		SemanticSummary: profile.SemanticSummary,
	}, nil
}

// UpdateCustomerTier updates the customer tier.
func (h *SynapseHandler) UpdateCustomerTier(ctx context.Context, req *pb.UpdateCustomerTierRequest) (*pb.UpdateCustomerTierResponse, error) {
	id, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id UUID")
	}
	err = h.svc.UpdateCustomerTier(ctx, id, domain.CustomerTier(req.Tier))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update customer tier: "+err.Error())
	}
	return &pb.UpdateCustomerTierResponse{
		Success: true,
	}, nil
}

// ListInteractionsByCustomer retrieves all interactions for a specific customer.
func (h *SynapseHandler) ListInteractionsByCustomer(ctx context.Context, req *pb.ListInteractionsByCustomerRequest) (*pb.ListInteractionsByCustomerResponse, error) {
	id, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id UUID")
	}
	logs, err := h.svc.ListInteractionsByCustomer(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list interactions: "+err.Error())
	}
	resp := &pb.ListInteractionsByCustomerResponse{
		Logs: make([]*pb.InteractionLogSummary, len(logs)),
	}
	for i, l := range logs {
		resp.Logs[i] = &pb.InteractionLogSummary{
			InteractionId: l.ID.String(),
			CustomerId:    l.CustomerID.String(),
			Channel:       string(l.Channel),
			InboundMsg:    l.InboundMsg,
			OutboundMsg:   l.OutboundMsg,
			TokensUsed:    l.TokensUsed,
			LatencyMs:     l.LatencyMs,
			CreatedAt:     l.CreatedAt.Format(time.RFC3339),
		}
	}
	return resp, nil
}


