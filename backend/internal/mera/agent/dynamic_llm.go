package agent

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/repository"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/genai"
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
// It implements model routing: if the primary model fails, it falls back to the next model.
func (d *DynamicLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	llms := d.resolveLLMs(ctx)

	return func(yield func(*model.LLMResponse, error) bool) {
		var lastErr error

		for _, llm := range llms {
			success := true
			req.Model = llm.Name() // update request model name

			next := llm.GenerateContent(ctx, req, stream)
			next(func(resp *model.LLMResponse, err error) bool {
				if err != nil {
					lastErr = err
					success = false
					slog.Warn("DynamicLLM: model generation failed, attempting routing fallback", "model", llm.Name(), "error", err)
					return false // stop this model's sequence
				}
				// Yield the successful response
				return yield(resp, nil)
			})

			// If success is true, this model completed without errors, stop routing
			if success {
				return
			}
		}

		if lastErr != nil {
			yield(nil, fmt.Errorf("all models in routing chain failed, last error: %w", lastErr))
		}
	}
}

// resolveLLMs finds the correct model.LLM chain to use for the current context.
func (d *DynamicLLM) resolveLLMs(ctx context.Context) []model.LLM {
	bizIDStr, err := repository.BusinessIDFromContext(ctx)
	if err != nil {
		slog.Warn("DynamicLLM: no business ID in context, using fallback LLM")
		return []model.LLM{d.fallbackLLM}
	}

	// 1. Check in-memory cache first
	if cached, ok := d.clients.Load(bizIDStr); ok {
		return cached.([]model.LLM)
	}

	// 2. Not in cache, check database for a custom key
	bizID, err := uuid.Parse(bizIDStr)
	if err != nil {
		return []model.LLM{d.fallbackLLM}
	}

	customKey, err := d.secretsRepo.GetSecret(ctx, bizID, domain.SecretKeyGeminiAPI)
	if err != nil {
		slog.Warn("DynamicLLM: failed to fetch custom key, using fallback", "business_id", bizIDStr, "error", err)
	} else if customKey != "" {
		// Try to build a routing chain of models
		fallbackModels := []string{d.modelName, "gemini-2.5-flash-lite", "gemini-2.0-flash"}
		var clients []model.LLM
		
		for _, mName := range fallbackModels {
			customClient, err := gemini.NewModel(ctx, mName, &genai.ClientConfig{
				APIKey: customKey,
			})
			if err == nil {
				clients = append(clients, customClient)
			}
		}
		
		if len(clients) > 0 {
			slog.Debug("DynamicLLM: instantiated custom Gemini routing chain for tenant", "business_id", bizIDStr, "chain_length", len(clients))
			d.clients.Store(bizIDStr, clients)
			return clients
		}
		
		slog.Error("DynamicLLM: failed to instantiate any custom Gemini clients, falling back", "business_id", bizIDStr)
	}

	// 3. Fallback
	return []model.LLM{d.fallbackLLM}
}

// EmbedContent generates a 768-dimensional embedding for the given text using the business's API key.
// It returns a slice of zeros if it fails, to gracefully degrade.
func (d *DynamicLLM) EmbedContent(ctx context.Context, text string) []float32 {
	bizIDStr, err := repository.BusinessIDFromContext(ctx)
	if err == nil {
		bizID, _ := uuid.Parse(bizIDStr)
		customKey, err := d.secretsRepo.GetSecret(ctx, bizID, domain.SecretKeyGeminiAPI)
		if err == nil && customKey != "" {
			client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: customKey})
			if err == nil {
				dim := int32(768)
				res, err := client.Models.EmbedContent(ctx, "gemini-embedding-2", genai.Text(text), &genai.EmbedContentConfig{
					OutputDimensionality: &dim,
				})
				if err == nil && len(res.Embeddings) > 0 {
					return res.Embeddings[0].Values
				}
			}
		}
	}
	// Fallback to dummy vector if embedding fails (768 dims)
	return make([]float32, 768)
}

// Ensure DynamicLLM implements model.LLM
var _ model.LLM = (*DynamicLLM)(nil)
