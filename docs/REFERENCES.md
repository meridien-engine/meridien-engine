# Meridien Engine — Architectural & Engineering References

This document catalogs key research repositories, mathematical models, framework specifications, and engineering resources utilized during the design and construction of Meridien Engine v2.0.

---

## 1. AI Reasoning & Workflow Core

### Google Agent Development Kit (ADK) v2
*   **Module Path**: `google.golang.org/adk/v2`
*   **Documentation & Specs**: Built-in packages for `workflow` and `runner`.
*   **Key Reference Concepts**:
    *   **Workflow Graphs**: Node-based directed graphs enforcing structured sequence loops rather than raw ReAct execution blocks.
    *   **State Suspensions**: Using `workflow.Suspend` and `RequestedInput` to allow human operators to review transaction details before order execution.
    *   **Session Management**: State synchronization across multi-turn sessions (e.g. `session.InMemoryService`).

### Arabic-English Semantic Chunker (LLM-Based)
*   **Source Repository**: [mohamed468/arabic-english-semantic-chunker](https://github.com/mohamed468/arabic-english-semantic-chunker)
*   **Key Design Insights**:
    *   Highlights the fragility of punctuation-based splitters for complex Arabic syntax, English technical structures, and code-switched retail models.
    *   Demonstrates how LLM fine-tuning can yield highly precise semantic boundaries for multi-language ingestion.
    *   We adapted this project's structural prompt formatting to run natively inside our monolithic `gemini-2.5-flash` client.

---

## 2. Mathematical Vector Models (RAG)

### Cosine Similarity & Distance
Utilized inside the pgvector RAG database repository and our statistical semantic chunking research profiles:
*   **Cosine Similarity Formula**:
    $$\text{Sim}(U, V) = \frac{U \cdot V}{\|U\| \|V\|}$$
*   **Cosine Distance Formula**:
    $$\text{Dist}(U, V) = 1 - \text{Sim}(U, V)$$

### Dynamic Breakpoint Thresholding
Used for distance-based chunk segmentation:
$$\text{Threshold} = \mu + k \cdot \sigma$$
*   $\mu$ = Mean of all adjacent distance gaps in the document.
*   $\sigma$ = Standard deviation of the distance gaps.
*   $k$ = Scaling constant (typically configured between $1.0$ and $1.2$).

---

## 3. Database Isolation & Tenancy

### PostgreSQL Row-Level Security (RLS)
*   **PostgreSQL Reference Docs**: [Row-Level Security Policies](https://www.postgresql.org/docs/current/ddl-rowsecurity.html).
*   **Session Variable Isolation**: Using `SET LOCAL app.current_business` to dynamically bind business tenant IDs inside transactions (`current_setting('app.current_business', true)::uuid`).

---

## 4. Observability & Tracing Infrastructure

### OpenTelemetry Go SDK & Tempo
*   **Framework**: [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/).
*   **Collector Specification**: Standard `otel-collector-config` shipping metrics to Prometheus and traces to Grafana Tempo.
*   **Grafana Tempo Docs**: [Tempo Distributed Tracing](https://grafana.com/docs/tempo/latest/).
