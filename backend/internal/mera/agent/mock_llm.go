package agent

import (
	"context"
	"iter"
	"strings"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

// MockLLM is a dummy implementation of model.LLM for testing and local dev.
type MockLLM struct{}

func (m *MockLLM) Name() string {
	return "mock-gemini-model"
}

func (m *MockLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		var prompt string
		if len(req.Contents) > 0 && len(req.Contents[0].Parts) > 0 {
			prompt = req.Contents[0].Parts[0].Text
		}

		replyText := "Mock Gemini reply"

		// Mock the intent router response
		if strings.Contains(prompt, "Classify the intent") {
			idx := strings.Index(prompt, "Analyze the customer's message")
			msg := ""
			if idx != -1 {
				sub := prompt[idx:]
				firstQuote := strings.Index(sub, "\"")
				if firstQuote != -1 {
					secondQuote := strings.Index(sub[firstQuote+1:], "\"")
					if secondQuote != -1 {
						msg = sub[firstQuote+1 : firstQuote+1+secondQuote]
					}
				}
			}
			msg = strings.ToLower(msg)
			if strings.Contains(msg, "buy") || strings.Contains(msg, "order") || strings.Contains(msg, "purchase") || strings.Contains(msg, "checkout") {
				replyText = `{"intent": "CHECKOUT", "skus": ["WIDGET-01"], "qtys": [1]}`
			} else {
				replyText = `{"intent": "INQUIRY"}`
			}
		} else if strings.Contains(prompt, "You are Mera") {
			// Mock the inquiry RAG response
			replyText = "Answer derived from mock knowledge sources: [Mock Gemini response using RAG]"
		} else if strings.Contains(prompt, "precise document segmentation") {
			// Mock the semantic chunker response
			replyText = `["Hello world.", "This is a semantic chunk test."]`
		}

		resp := &model.LLMResponse{
			Content: &genai.Content{
				Role: "model",
				Parts: []*genai.Part{
					genai.NewPartFromText(replyText),
				},
			},
		}

		yield(resp, nil)
	}
}

// Ensure MockLLM implements model.LLM
var _ model.LLM = &MockLLM{}
