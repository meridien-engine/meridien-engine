// Package grpchandler — Knowledge gRPC handler.
package grpchandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/gen/knowledge"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// KnowledgeHandler implements the gRPC KnowledgeService server interface.
type KnowledgeHandler struct {
	knowledge.UnimplementedKnowledgeServiceServer
	repo      domain.KnowledgeRepository
	embedFunc EmbedFunc
	llmModel  model.LLM
}

// EmbedFunc converts a text string into a float32 embedding vector.
// The implementation calls the configured embedding API (e.g. OpenAI).
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

func NewKnowledgeHandler(repo domain.KnowledgeRepository, embed EmbedFunc, llmModel model.LLM) *KnowledgeHandler {
	return &KnowledgeHandler{repo: repo, embedFunc: embed, llmModel: llmModel}
}

// QueryKnowledge performs a vector similarity search for Mera's RAG step.
func (h *KnowledgeHandler) QueryKnowledge(ctx context.Context, req *knowledge.KnowledgeQuery) (*knowledge.KnowledgeResult, error) {
	if req.QueryText == "" {
		return nil, status.Error(codes.InvalidArgument, "query_text is required")
	}

	topK := int(req.TopK)
	if topK <= 0 {
		topK = 5 // sensible default
	}

	// Embed the query text.
	embedding, err := h.embedFunc(ctx, req.QueryText)
	if err != nil {
		return nil, status.Error(codes.Internal, "embedding failed: "+err.Error())
	}

	// Time the vector search for Prometheus.
	start := time.Now()
	chunks, err := h.repo.Query(ctx, embedding, topK)
	elapsed := time.Since(start).Seconds()

	metrics.RAGQueryDuration.WithLabelValues(fmt.Sprint(topK)).Observe(elapsed)
	if err != nil {
		return nil, status.Error(codes.Internal, "vector search failed: "+err.Error())
	}

	metrics.RAGChunksReturned.Observe(float64(len(chunks)))

	resp := &knowledge.KnowledgeResult{
		Chunks: make([]*knowledge.KnowledgeChunk, len(chunks)),
	}
	for i, c := range chunks {
		resp.Chunks[i] = &knowledge.KnowledgeChunk{
			NodeId:  c.NodeID.String(),
			Source:  c.Source,
			Content: c.Content,
			Score:   c.Score,
		}
	}
	return resp, nil
}

// IngestDocument chunks, embeds, and stores a merchant document.
// Called after a merchant uploads a PDF or FAQ from the portal.
func (h *KnowledgeHandler) IngestDocument(ctx context.Context, req *knowledge.IngestRequest) (*knowledge.IngestResponse, error) {
	if req.SourceName == "" || req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "source_name and content are required")
	}

	// Semantic LLM-Based Chunking using h.llmModel
	chunks, err := h.semanticChunkText(ctx, req.Content)
	if err != nil {
		// Fallback to naive character-based chunking if LLM chunking fails
		chunks = chunkText(req.Content, 512, 64)
	}

	created := 0
	for _, chunk := range chunks {
		embedding, err := h.embedFunc(ctx, chunk)
		if err != nil {
			return nil, status.Error(codes.Internal, "embedding failed for chunk: "+err.Error())
		}
		if err := h.repo.InsertChunk(ctx, req.SourceName, chunk, embedding); err != nil {
			return nil, status.Error(codes.Internal, "insert chunk failed: "+err.Error())
		}
		created++
	}

	return &knowledge.IngestResponse{ChunksCreated: int32(created), Success: true}, nil
}

// semanticChunkText segments raw text into semantic chunks using the configured LLM.
func (h *KnowledgeHandler) semanticChunkText(ctx context.Context, text string) ([]string, error) {
	if h.llmModel == nil {
		return nil, fmt.Errorf("LLM model not configured for semantic chunking")
	}

	prompt := fmt.Sprintf(`You are a precise document segmentation agent. Your task is to split the input text into semantically coherent chunks based on meaning, topic, and paragraph boundaries. This system supports English, Arabic, and code-switched technical files. You MUST NOT summarize, change, translate, or delete any characters from the original text. Output strictly a JSON array of strings containing the chunks in order.

Segment the following text into logical chunks:

[TEXT START]
%s
[TEXT END]`, text)

	req := &model.LLMRequest{
		Model: h.llmModel.Name(),
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					genai.NewPartFromText(prompt),
				},
			},
		},
		Config: &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
		},
	}

	var replyJSON string
	for resp, err := range h.llmModel.GenerateContent(ctx, req, false) {
		if err != nil {
			return nil, err
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			replyJSON += resp.Content.Parts[0].Text
		}
	}

	var chunks []string
	if err := json.Unmarshal([]byte(replyJSON), &chunks); err != nil {
		return nil, err
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("LLM returned empty chunks")
	}

	return chunks, nil
}

// ─── chunking helper ──────────────────────────────────────────────────────────

// chunkText splits text into overlapping windows of 'size' runes with 'overlap' rune overlap.
func chunkText(text string, size, overlap int) []string {
	runes := []rune(text)
	var chunks []string
	for start := 0; start < len(runes); start += size - overlap {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}
