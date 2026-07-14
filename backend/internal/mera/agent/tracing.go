// Package agent implements the Mera ADK workflow graph nodes and the
// OpenTelemetry instrumentation layer that wraps them.
package agent

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/meridien-engine/meridien-engine/internal/repository"
)

// ─── Required span names ───────────────────────────────────────────────────
// Every ADK workflow node MUST be wrapped with WithTrace using one of these
// constants. Adding a new node requires adding a constant here first.

const (
	SpanResolveCustomer = "mera.resolve_customer"
	SpanRAGRetrieval    = "mera.rag_retrieval"
	SpanERPCheckout     = "mera.erp_checkout"
	SpanLLMRoute        = "mera.llm_route"
	SpanHITLSuspend     = "mera.hitl_suspend"
)

// ─── NodeFn is the function signature for all Mera workflow nodes ──────────
// ADK v2 node functions will satisfy this interface; the wrapper below is
// type-compatible with workflow.NewFunctionNode's expected signature.
// Replace with the actual ADK type alias once the module is imported.
type NodeFn func(ctx context.Context, input map[string]any) (map[string]any, error)

// ─── WithTrace wraps any NodeFn with an OpenTelemetry span ────────────────
//
// Usage:
//
//	node := workflow.NewFunctionNode("rag", WithTrace(SpanRAGRetrieval, ragFn))
//
// Every span automatically carries:
//   - "business_id" attribute from context (for per-tenant trace filtering)
//   - error recording + span status on failure
func WithTrace(spanName string, fn NodeFn) NodeFn {
	return func(ctx context.Context, input map[string]any) (map[string]any, error) {
		ctx, span := otel.Tracer("mera").Start(ctx, spanName)
		defer span.End()

		// Propagate tenant ID so traces can be filtered per-merchant in Grafana.
		if bID, err := repository.BusinessIDFromContext(ctx); err == nil {
			span.SetAttributes(attribute.String("business_id", bID))
		}

		out, err := fn(ctx, input)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
		}
		return out, err
	}
}

// ─── InitTracer bootstraps the global OTEL tracer provider ───────────────
//
// Call this once from main.go before starting the HTTP/gRPC servers.
// otlpEndpoint example: "localhost:4317" (Jaeger / OTEL collector gRPC port).
//
// The returned shutdown function must be deferred in main to flush spans:
//
//	shutdown, err := agent.InitTracer(ctx, cfg.OTLPEndpoint)
//	if err != nil { ... }
//	defer shutdown(ctx)
func InitTracer(ctx context.Context, otlpEndpoint string) (func(context.Context) error, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(), // TLS is terminated at the collector in prod
	)
	if err != nil {
		return nil, fmt.Errorf("mera tracer: create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("meridien-mera"),
			semconv.ServiceVersion("v1"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("mera tracer: create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		// Sample 100% in dev; override via OTEL_TRACES_SAMPLER env var in prod.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
