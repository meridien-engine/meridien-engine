// Package brain implements the Mera AI agent orchestrator.
//
// Mera follows a strict ReAct loop:
//   Thought → Action (Tool Call) → Observation → ... → Final Response
//
// The agent has access to three tools, each backed by a gRPC client:
//   - synapse_lookup   : Resolves customer UCM + semantic summary
//   - knowledge_search : Retrieves relevant knowledge chunks (RAG)
//   - place_order      : Submits an order to the ERP service
//
// Mera never handles pricing — that invariant is enforced by the ERP service.
package brain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
)

// Tool is a callable capability exposed to the Mera reasoning loop.
type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, input string) (string, error)
}

// LLMClient is the interface for communicating with the underlying language model.
// Abstracting this allows swapping providers (Anthropic Claude, Google Gemini, etc.)
// without changing the agent logic.
type LLMClient interface {
	// Complete sends a prompt and returns the model's text output.
	Complete(ctx context.Context, systemPrompt, userMessage string) (string, int32, error)
}

// Agent is the Mera orchestrator. It resolves customer context, retrieves
// relevant knowledge, runs the ReAct loop, and records the full trace.
type Agent struct {
	llm      LLMClient
	synapse  domain.CustomerRepository
	interact domain.InteractionRepository
	tools    map[string]Tool
}

// NewAgent constructs a Mera agent with the provided tools registered.
func NewAgent(llm LLMClient, customers domain.CustomerRepository, interact domain.InteractionRepository, tools []Tool) *Agent {
	tm := make(map[string]Tool, len(tools))
	for _, t := range tools {
		tm[t.Name()] = t
	}
	return &Agent{llm: llm, synapse: customers, interact: interact, tools: tm}
}

// Turn processes a single customer message and returns Mera's response.
// It records the full interaction trace for Compass observability.
func (a *Agent) Turn(ctx context.Context, req TurnRequest) (*TurnResponse, error) {
	start := time.Now()

	// ── Step 1: Resolve customer UCM ─────────────────────────────────────────
	profile, _, err := a.synapse.GetOrCreateByChannel(ctx, req.Channel, req.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("synapse lookup: %w", err)
	}

	// ── Step 2: Build system prompt injecting UCM context ────────────────────
	systemPrompt := buildSystemPrompt(profile, a.tools)

	// ── Step 3: ReAct loop (max 5 iterations to prevent runaway loops) ───────
	var (
		thoughts  string
		toolCalls []domain.ToolCall
		contexts  []domain.RetrievedContext
		response  string
		tokens    int32
	)

	loopMsg := req.Message
	for i := 0; i < 5; i++ {
		raw, toks, err := a.llm.Complete(ctx, systemPrompt, loopMsg)
		tokens += toks
		if err != nil {
			return nil, fmt.Errorf("llm complete (iter %d): %w", i, err)
		}
		thoughts += raw + "\n"

		action, done := parseAction(raw)
		if done {
			response = parseAnswer(raw)
			break
		}

		tool, ok := a.tools[action.ToolName]
		if !ok {
			loopMsg = fmt.Sprintf("Observation: tool %q does not exist.", action.ToolName)
			continue
		}

		result, err := tool.Run(ctx, action.Input)
		if err != nil {
			loopMsg = fmt.Sprintf("Observation: tool %q returned error: %v", action.ToolName, err)
		} else {
			loopMsg = fmt.Sprintf("Observation: %s", result)
		}

		toolCalls = append(toolCalls, domain.ToolCall{
			ToolName:   action.ToolName,
			ArgsJSON:   action.Input,
			ResultJSON: result,
		})
	}

	if response == "" {
		response = "I'm sorry, I wasn't able to complete your request. Please try again."
	}

	// ── Step 4: Record trace ──────────────────────────────────────────────────
	latency := int32(time.Since(start).Milliseconds())
	logID := uuid.New()

	log := &domain.InteractionLog{
		ID:          logID,
		BusinessID:  profile.BusinessID,
		CustomerID:  profile.ID,
		Channel:     req.Channel,
		InboundMsg:  req.Message,
		OutboundMsg: response,
		TokensUsed:  tokens,
		LatencyMs:   latency,
	}

	trace := &domain.InteractionTrace{
		ID:                uuid.New(),
		InteractionLogID:  logID,
		RetrievedContexts: contexts,
		SystemPrompt:      systemPrompt,
		RawAgentThoughts:  thoughts,
		ToolsCalled:       toolCalls,
	}

	if err := a.interact.RecordTurn(ctx, log, trace); err != nil {
		// Non-fatal — log the error but still return the response.
		// We prefer delivering the answer over failing the customer turn.
		_ = err // TODO: structured log via slog
	}

	return &TurnResponse{
		Message:       response,
		InteractionID: logID.String(),
	}, nil
}

// ─── Supporting types ─────────────────────────────────────────────────────────

// TurnRequest is the input to a single Mera conversation turn.
type TurnRequest struct {
	Channel    domain.ChannelType
	ExternalID string // Platform-specific identifier (phone number, user ID)
	Message    string
}

// TurnResponse is Mera's output from a single turn.
type TurnResponse struct {
	Message       string
	InteractionID string
}

// actionCall represents a parsed tool invocation from the LLM output.
type actionCall struct {
	ToolName string
	Input    string
}

// ─── Prompt & parsing helpers (stubs — expanded in brain/prompt.go) ──────────

func buildSystemPrompt(profile *domain.CustomerProfile, tools map[string]Tool) string {
	// TODO: implement rich prompt template in brain/prompt.go
	_ = profile
	_ = tools
	return "You are Mera, a helpful retail assistant."
}

func parseAction(raw string) (actionCall, bool) {
	// TODO: implement in brain/parser.go
	_ = raw
	return actionCall{}, true // stub: treat every output as final answer
}

func parseAnswer(raw string) string {
	return raw // stub: return full output as answer
}
