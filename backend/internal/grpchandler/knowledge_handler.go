// Package grpchandler — Knowledge gRPC handler.
package grpchandler

import (
	"context"
	"fmt"
	"time"

	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// KnowledgeHandler implements the gRPC KnowledgeService server interface.
type KnowledgeHandler struct {
	repo      domain.KnowledgeRepository
	embedFunc EmbedFunc
}

// EmbedFunc converts a text string into a float32 embedding vector.
// The implementation calls the configured embedding API (e.g. OpenAI).
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

func NewKnowledgeHandler(repo domain.KnowledgeRepository, embed EmbedFunc) *KnowledgeHandler {
	return &KnowledgeHandler{repo: repo, embedFunc: embed}
}

// QueryKnowledge performs a vector similarity search for Mera's RAG step.
func (h *KnowledgeHandler) QueryKnowledge(ctx context.Context, req *QueryKnowledgeRequest) (*QueryKnowledgeResponse, error) {
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

	resp := &QueryKnowledgeResponse{
		Chunks: make([]*KnowledgeChunkDTO, len(chunks)),
	}
	for i, c := range chunks {
		resp.Chunks[i] = &KnowledgeChunkDTO{
			NodeID:  c.NodeID.String(),
			Source:  c.Source,
			Content: c.Content,
			Score:   c.Score,
		}
	}
	return resp, nil
}

// IngestDocument chunks, embeds, and stores a merchant document.
// Called after a merchant uploads a PDF or FAQ from the portal.
func (h *KnowledgeHandler) IngestDocument(ctx context.Context, req *IngestDocumentRequest) (*IngestDocumentResponse, error) {
	if req.SourceName == "" || req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "source_name and content are required")
	}

	// Naive chunking: split into ~512-character windows with 64-char overlap.
	chunks := chunkText(req.Content, 512, 64)
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

	return &IngestDocumentResponse{ChunksCreated: int32(created), Success: true}, nil
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

// ─── DTO types ────────────────────────────────────────────────────────────────

type QueryKnowledgeRequest struct {
	QueryText string
	TopK      int32
}

type KnowledgeChunkDTO struct {
	NodeID  string
	Source  string
	Content string
	Score   float64
}

type QueryKnowledgeResponse struct {
	Chunks []*KnowledgeChunkDTO
}

type IngestDocumentRequest struct {
	SourceName string
	Content    string
}

type IngestDocumentResponse struct {
	ChunksCreated int32
	Success       bool
}
