// Package agent implements the Mera AI agent workflow graph, nodes,
// and execution hooks.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/workflowagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/workflow"
	"google.golang.org/genai"
)

// CustomerResolutionOutput represents the state after resolving the customer.
type CustomerResolutionOutput struct {
	Message  string                  `json:"message"`
	Customer *domain.CustomerProfile `json:"customer"`
	IsNew    bool                    `json:"is_new"`
}

// RAGRetrievalOutput represents the enriched state after RAG vector search.
type RAGRetrievalOutput struct {
	CustomerOutput CustomerResolutionOutput `json:"customer_output"`
	Chunks         []domain.KnowledgeChunk  `json:"chunks"`
}

// RouterOutput carries the parsed checkout intent details.
type RouterOutput struct {
	RAGOutput RAGRetrievalOutput `json:"rag_output"`
	SKUs      []string           `json:"skus"`
	Qtys      []int32            `json:"qtys"`
}

// LLMRouterResponse holds the structured JSON output from the Gemini intent classification.
type LLMRouterResponse struct {
	Intent string   `json:"intent"`
	SKUs   []string `json:"skus"`
	Qtys   []int32  `json:"qtys"`
}

// mockEmbedding generates a dummy 1536-dimensional embedding slice for local development.

// NewMeraWorkflow constructs and wires the ADK workflow graph for Mera reasoning loops.
func NewMeraWorkflow(
	llmModel model.LLM,
	synSvc *synapse.Service,
	erpSvc *erp.Service,
	productRepo domain.ProductRepository,
	knowledgeRepo domain.KnowledgeRepository,
) (agent.Agent, error) {

	// ── Node 1: Resolve Customer Profile ──────────────────────────────────────
	resolveCustomerNode := workflow.NewFunctionNode("resolve_customer",
		func(ctx agent.Context, in string) (CustomerResolutionOutput, error) {
			channelVal, err := ctx.Session().State().Get("channel")
			if err != nil {
				return CustomerResolutionOutput{}, fmt.Errorf("resolve customer node: missing channel state: %w", err)
			}
			channel, _ := channelVal.(string)

			externalIDVal, err := ctx.Session().State().Get("channel_external_id")
			if err != nil {
				return CustomerResolutionOutput{}, fmt.Errorf("resolve customer node: missing channel_external_id state: %w", err)
			}
			externalID, _ := externalIDVal.(string)

			profile, isNew, err := synSvc.GetOrCreateCustomer(
				ctx,
				domain.ChannelType(channel),
				externalID,
			)
			if err != nil {
				return CustomerResolutionOutput{}, fmt.Errorf("resolve customer node: %w", err)
			}
			return CustomerResolutionOutput{
				Message:  in,
				Customer: profile,
				IsNew:    isNew,
			}, nil
		},
		workflow.NodeConfig{},
	)

	// ── Node 2: RAG Vector Retrieval ──────────────────────────────────────────
	ragRetrievalNode := workflow.NewFunctionNode("rag_retrieval",
		func(ctx agent.Context, in CustomerResolutionOutput) (RAGRetrievalOutput, error) {
			var emb []float32
			if dynLLM, ok := llmModel.(*DynamicLLM); ok {
				emb = dynLLM.EmbedContent(ctx, in.Message)
			} else {
				// Fallback to zeros if not DynamicLLM
				emb = make([]float32, 768)
			}
			
			chunks, err := knowledgeRepo.Query(ctx, emb, 3)
			if err != nil {
				return RAGRetrievalOutput{
					CustomerOutput: in,
					Chunks:         []domain.KnowledgeChunk{},
				}, nil
			}
			return RAGRetrievalOutput{
				CustomerOutput: in,
				Chunks:         chunks,
			}, nil
		},
		workflow.NodeConfig{},
	)

	// ── Node 3: Router LLM Node (Intents: Checkout vs Inquiry) ────────────────
	routerNode := workflow.NewFunctionNode("intent_router",
		func(ctx agent.Context, in RAGRetrievalOutput) (*session.Event, error) {
			ev := session.NewEvent(ctx, ctx.InvocationID())
			msg := in.CustomerOutput.Message

			products, _ := productRepo.List(ctx)
			var catalogStr string
			for _, p := range products {
				catalogStr += fmt.Sprintf("- %s (SKU: %s, Price: $%s)\n", p.Name, p.SKU, p.Price.String())
			}
			if catalogStr == "" {
				catalogStr = "No products available."
			}

			prompt := fmt.Sprintf(`You are the router for Meridien Engine. Analyze the customer's message:
%q

Available Product Catalog:
%s

Classify the intent into one of:
- "CHECKOUT": If they want to purchase or buy one or more products from the catalog.
- "INQUIRY": If they are asking a question about a product, price, shipping, policies, or general questions.

Respond ONLY with a JSON object in this format. For "skus", you MUST use the exact SKU from the Available Product Catalog above:
{
  "intent": "CHECKOUT" | "INQUIRY",
  "skus": ["SKU1", "SKU2"],
  "qtys": [1, 2]
}
`, msg, catalogStr)

			req := &model.LLMRequest{
				Model: llmModel.Name(),
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
			for resp, err := range llmModel.GenerateContent(ctx, req, false) {
				if err != nil {
					return nil, fmt.Errorf("gemini router generation failed: %w", err)
				}
				if resp.Content != nil && len(resp.Content.Parts) > 0 {
					replyJSON += resp.Content.Parts[0].Text
				}
			}

			var lr LLMRouterResponse
			if err := json.Unmarshal([]byte(replyJSON), &lr); err != nil {
				// Fallback to INQUIRY if JSON unmarshal fails
				ev.Routes = []string{"INQUIRY"}
				ev.Output = in
				return ev, nil
			}

			if lr.Intent == "CHECKOUT" {
				ev.Routes = []string{"CHECKOUT"}
				ev.Output = RouterOutput{
					RAGOutput: in,
					SKUs:      lr.SKUs,
					Qtys:      lr.Qtys,
				}
			} else {
				ev.Routes = []string{"INQUIRY"}
				ev.Output = in
			}

			return ev, nil
		},
		workflow.NodeConfig{},
	)

	// ── Node 4: ERP Transaction Node (with Rerun-on-Resume HITL suspension) ───
	rerun := true
	erpCheckoutNode := workflow.NewEmittingFunctionNode("erp_checkout",
		func(ctx agent.Context, in RouterOutput, emit func(*session.Event) error) (any, error) {
			var catalogTotal float64
			var lines []domain.OrderLine
			for i, sku := range in.SKUs {
				qty := in.Qtys[i]
				lines = append(lines, domain.OrderLine{SKU: sku, Quantity: qty})

				p, err := productRepo.GetBySKU(ctx, sku)
				if err == nil && p != nil {
					catalogTotal += p.Price.InexactFloat64() * float64(qty)
				}
			}

			var expectedPrice float64
			if expectedVal, err := ctx.Session().State().Get("expected_price"); err == nil {
				expectedPrice, _ = expectedVal.(float64)
			}

			if expectedPrice > 0 && expectedPrice < catalogTotal {
				reply, err := workflow.ResumeOrRequestInput(ctx, emit, session.RequestInput{
					InterruptID: "price_discrepancy-" + ctx.InvocationID(),
					Message:     fmt.Sprintf("Price mismatch: Customer expected $%g, catalog is $%g. Approve?", expectedPrice, catalogTotal),
				})
				if err != nil {
					return nil, err
				}

				replyStr, _ := reply.(string)
				if replyStr != "approved" {
					return "Checkout cancelled: price adjustment rejected by merchant.", nil
				}
			}

			cmd := domain.PlaceOrderCommand{
				CustomerID: in.RAGOutput.CustomerOutput.Customer.ID,
				Source:     domain.OrderSourceAgent,
				Notes:      in.RAGOutput.CustomerOutput.Message,
				Lines:      lines,
			}
			order, err := erpSvc.PlaceOrder(ctx, cmd)
			if err != nil {
				return nil, fmt.Errorf("checkout order: %w", err)
			}

			return fmt.Sprintf("Order placed successfully! Total: %s", order.TotalPrice.String()), nil
		},
		workflow.NodeConfig{RerunOnResume: &rerun},
	)

	// ── Node 5: Inquiry Handler Node ──────────────────────────────────────────
	inquiryNode := workflow.NewFunctionNode("inquiry",
		func(ctx agent.Context, in RAGRetrievalOutput) (string, error) {
			var contexts []string
			for _, chunk := range in.Chunks {
				contexts = append(contexts, fmt.Sprintf("Source: %s\nContent: %s", chunk.Source, chunk.Content))
			}
			contextStr := strings.Join(contexts, "\n\n")

			prompt := fmt.Sprintf(`You are Mera, a helpful AI customer representative for Meridien Engine.
Answer the customer's question politely and accurately using ONLY the provided knowledge sources.
If the answer is not in the knowledge sources, say politely that you don't know or ask them to contact support.

Customer message: %q

Knowledge sources:
%s
`, in.CustomerOutput.Message, contextStr)

			req := &model.LLMRequest{
				Model: llmModel.Name(),
				Contents: []*genai.Content{
					{
						Role: "user",
						Parts: []*genai.Part{
							genai.NewPartFromText(prompt),
						},
					},
				},
			}

			var replyText string
			for resp, err := range llmModel.GenerateContent(ctx, req, false) {
				if err != nil {
					return "", fmt.Errorf("gemini inquiry generation failed: %w", err)
				}
				if resp.Content != nil && len(resp.Content.Parts) > 0 {
					replyText += resp.Content.Parts[0].Text
				}
			}

			return replyText, nil
		},
		workflow.NodeConfig{},
	)

	// ── 6. Assemble Workflow Graph Edges ──────────────────────────────────────
	edges := workflow.NewEdgeBuilder().
		Add(workflow.Start, resolveCustomerNode).
		Add(resolveCustomerNode, ragRetrievalNode).
		Add(ragRetrievalNode, routerNode).
		AddRoute(routerNode, erpCheckoutNode, workflow.StringRoute("CHECKOUT")).
		AddRoute(routerNode, inquiryNode, workflow.StringRoute("INQUIRY")).
		Build()

	// ── 7. Build and return the workflow agent ────────────────────────────────
	wfAgent, err := workflowagent.New(workflowagent.Config{
		Name:        "mera_workflow",
		Description: "Directed graph workflow governing Mera client messaging",
		Edges:       edges,
	})
	if err != nil {
		return nil, fmt.Errorf("new workflow agent: %w", err)
	}

	return wfAgent, nil
}
