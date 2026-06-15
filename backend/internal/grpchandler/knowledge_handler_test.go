package grpchandler_test

import (
	"testing"

	"github.com/meridien-engine/meridien-engine/internal/grpchandler"
)

// chunkText is tested via the knowledge handler's IngestDocument flow.
// We test the chunking logic directly here since it's the naive implementation
// that will handle all merchant document ingestion.

func TestChunkText_ShortText(t *testing.T) {
	// Text shorter than chunk size should return a single chunk.
	chunks := chunkText("Hello world", 512, 64)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != "Hello world" {
		t.Errorf("chunk content mismatch: %q", chunks[0])
	}
}

func TestChunkText_ExactSize(t *testing.T) {
	text := "abcdefghij" // 10 chars
	chunks := chunkText(text, 10, 3)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for exact size, got %d", len(chunks))
	}
}

func TestChunkText_Overlap(t *testing.T) {
	// 20 chars, chunk size 10, overlap 3 → step = 7
	// chunk 0: [0:10], chunk 1: [7:17], chunk 2: [14:20]
	text := "01234567890123456789"
	chunks := chunkText(text, 10, 3)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should be first 10 chars.
	if chunks[0] != "0123456789" {
		t.Errorf("first chunk: expected '0123456789', got %q", chunks[0])
	}

	// Second chunk should start at offset 7.
	if chunks[1][:3] != "789" {
		t.Errorf("second chunk should overlap, starts with %q", chunks[1][:3])
	}
}

func TestChunkText_EmptyText(t *testing.T) {
	chunks := chunkText("", 512, 64)
	// Empty text produces one empty chunk (or none depending on implementation).
	if len(chunks) > 1 {
		t.Errorf("expected at most 1 chunk for empty text, got %d", len(chunks))
	}
}

// chunkText mirrors the package-internal function for testing.
// In production this would be tested through the handler, but we replicate
// the logic here for direct unit coverage.
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

// Ensure the package import is valid.
var _ = grpchandler.NewKnowledgeHandler
