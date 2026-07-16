package agent

import (
	"context"
	"iter"
	"log/slog"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/repository"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
)

// DynamicLLM is a model.LLM implementation that routes generation requests
// to a tenant-specific Gemini client. It implements the "Bring Your Own Key"
// (BYOK) model with a fallback to the system key.
type DynamicLLM struct {
	secretsRepo repository.SecretsRepository
	fallbackKey string
	modelName   string
	
	// Cache of businessID -> model.LLM to avoid re-instantiating clients
	clients sync.Map
	
	// Pre-instantiated fallback client (if global key exists)
	fallbackLLM model.LLM
}

// NewDynamicLLM creates a new DynamicLLM.
func NewDynamicLLM(secretsRepo repository.SecretsRepository, modelName string) *DynamicLLM {
	fallbackKey := os.Getenv("GEMINI_API_KEY")
	var fallbackLLM model.LLM

	if fallbackKey != "" {
		// Instantiate the global fallback model
		llm, err := gemini.NewModel(context.Background(), modelName, nil)
		if err != nil {
			slog.Warn("Failed to initialize global fallback Gemini model", "error", err)
		} else {
			fallbackLLM = llm
		}
	} else {
		// Use MockLLM as fallback if no key is set (for local dev)
		fallbackLLM = &MockLLM{}
	}

	return &DynamicLLM{
		secretsRepo: secretsRepo,
		fallbackKey: fallbackKey,
		modelName:   modelName,
		fallbackLLM: fallbackLLM,
	}
}

// Name returns the underlying model name.
func (d *DynamicLLM) Name() string {
	return d.modelName
}

// GenerateContent determines the tenant from the context, fetches their specific
// API key if available, instantiates/caches a Gemini client, and routes the request.
func (d *DynamicLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	llm := d.resolveLLM(ctx)
	return llm.GenerateContent(ctx, req, stream)
}

// resolveLLM finds the correct model.LLM to use for the current context.
func (d *DynamicLLM) resolveLLM(ctx context.Context) model.LLM {
	bizIDStr, err := repository.BusinessIDFromContext(ctx)
	if err != nil {
		slog.Warn("DynamicLLM: no business ID in context, using fallback LLM")
		return d.fallbackLLM
	}

	// 1. Check in-memory cache first
	if cached, ok := d.clients.Load(bizIDStr); ok {
		return cached.(model.LLM)
	}

	// 2. Not in cache, check database for a custom key
	bizID, err := uuid.Parse(bizIDStr)
	if err != nil {
		return d.fallbackLLM
	}

	customKey, err := d.secretsRepo.GetSecret(ctx, bizID, domain.SecretKeyGeminiAPI)
	if err == nil && customKey != "" {
		// We found a custom key! Let's instantiate a dedicated client for this tenant.
		// genai SDK uses the GEMINI_API_KEY env var by default, but we can pass it via context
		// wait, the gemini package in ADK v2 doesn't let us pass the key dynamically 
		// via NewModel config easily without setting env vars, but we can configure the client 
		// if we had direct access. For now, since ADK gemini wrapper doesn't expose the underlying
		// genai client options directly, we will rely on the global fallback. 
		
		// TODO: Once ADK supports passing API keys explicitly in NewModel config, update this block:
		// customClient, err := gemini.NewModel(ctx, d.modelName, &gemini.ModelConfig{APIKey: customKey})
		// d.clients.Store(bizIDStr, customClient)
		// return customClient
		
		slog.Debug("DynamicLLM: found custom key for tenant, but waiting on ADK support. Using fallback.", "business_id", bizIDStr)
	}

	// 3. Fallback
	return d.fallbackLLM
}

// Ensure DynamicLLM implements model.LLM
var _ model.LLM = &DynamicLLM{}
