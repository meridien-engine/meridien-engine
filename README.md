# Meridien Engine

A next-generation multi-tenant enterprise retail and customer intelligence platform — combining a high-performance ERP core with an AI reasoning layer that lets merchants understand their customers deeply and serve them intelligently, without giving the AI unsupervised access to business data.

---

## Prior Work

This project is a ground-up rewrite of [MERIDIEN](https://github.com/meridien-engine/MERIDIEN), the original ERP/CRM system that informed this architecture. The old repository serves as a reference for decisions made and lessons learned — not as a codebase to migrate from.

---

## Architecture Overview

Meridien Engine is a **Modular Monolith in Go** with four internal service boundaries communicating via gRPC:

| Service | Role | Package |
|---|---|---|
| **ERP Core** | Product catalog, orders, stock management, tenant isolation | `internal/erp` |
| **Synapse** | Unified Customer Model (UCM), identity resolution across channels | `internal/synapse` |
| **Knowledge** | pgvector RAG — semantic search over merchant-uploaded documents | `internal/repository` (knowledge) |
| **Mera Brain** | AI agent ReAct loop, tool dispatch, full trace recording | `internal/brain` |

### System Topology Overview

Meridien Engine is architected as a **Modular Monolith in Go**. High performance and loose coupling are enforced via **internal gRPC communication channels** between isolated functional layers, while external entry points are handled via REST/Webhooks (for user interfaces and messaging channels).

```text
 ┌────────────────────────────────────────────────────────────────────────┐  
 │                           EXTERNAL CLIENTS                             │  
 └───────┬───────────────────────────┬────────────────────────────┬───────┘  
         │ (HTTP REST)               │ (DMs / Webhooks)           │ (HTTP REST)  
         ▼                           ▼                            ▼  
 ┌───────────────┐           ┌───────────────┐            ┌───────────────┐  
 │ Meridien ERP  │           │  Mera Agent   │            │ Compass Dash  │  
 │   Frontend    │           │ (Chat/Social) │            │ (Analytics)   │  
 └───────┬───────┘           └───────┬───────┘            └───────┬───────┘  
         │                           │                            │  
 ════════╪═══════════════════════════╪════════════════════════════╪════════  
         │                           ▼                            │  
         │                   ┌───────────────┐                    │  
         │                   │  Mera Brain   │                    │  
         │                   │(Orchestrator) │                    │  
         │                   └─┬─────┬─────┬─┘                    │  
         │                     │     │     │                      │  
         │              (gRPC) │     │     │ (gRPC)               │  
         ▼                     ▼     │     ▼                      ▼  
 ┌───────────────┐ ◄───────────┘     │   ┌───────────────┐ ◄──────┴───────┐  
 │   ERP Core    │                   │   │    Synapse    │                │  
 │   Service     │ ◄─────────────────┼──►│    Service    │ (Co-located)   │  
 │               │                   │   │   & Compass   │                │  
 └───────┬───────┘                   │   └───────┬───────┘ ◄──────────────┘  
         │                           │           │  
         ▼                           ▼           ▼  
 ┌───────────────┐           ┌───────────────┐ ┌───────────────┐  
 │ Postgres DB  │           │   Knowledge   │ │ Postgres DB  │  
 │ (Tenant RLS)  │           │  Service (RAG)│ │  (Profiles)   │  
 └───────────────┘           └───────┬───────┘ └───────────────┘  
                                     ▼  
                             ┌───────────────┐  
                             │   Vector DB   │  
                             │ (Embeddings)  │  
                             └───────────────┘
```

Key design invariants:

- **Multi-tenant isolation** is enforced at the PostgreSQL layer via Row-Level Security (`app.current_business` UUID). Every write goes through `ExecWithTenant`.
- **Zero-hallucination pricing** — the AI agent submits SKU + quantity only. Price is always resolved from the database catalog by the ERP service.
- **Full AI observability** — every Mera turn records the system prompt, retrieved knowledge chunks, agent thoughts, and tool calls into `interaction_traces` for the Compass dashboard.

---

## Repository Structure

```
meridien-engine/
├── backend/                        # Go backend (modular monolith)
│   ├── api/proto/                  # gRPC Protobuf contracts
│   │   ├── orders.proto            # ERP OrderService
│   │   ├── synapse.proto           # SynapseService (UCM + traces)
│   │   └── knowledge.proto         # KnowledgeService (RAG)
│   ├── cmd/server/                 # Entry point + dependency injection
│   ├── internal/
│   │   ├── domain/                 # Pure entities, interfaces, errors (zero deps)
│   │   ├── erp/                    # ERP service (order placement, stock)
│   │   ├── synapse/                # Synapse service (UCM, trace recording)
│   │   ├── brain/                  # Mera AI agent (ReAct loop)
│   │   ├── repository/             # PostgreSQL implementations + RLS middleware
│   │   ├── grpchandler/            # gRPC transport handlers
│   │   ├── health/                 # /healthz and /readyz endpoints
│   │   ├── metrics/                # Prometheus instrumentation
│   │   └── db/                     # sqlc-generated database access layer
│   ├── db/
│   │   ├── migrations/             # golang-migrate SQL migrations (4 files)
│   │   └── queries/                # sqlc SQL query definitions
│   └── Dockerfile
│
├── infra/
│   ├── prometheus/prometheus.yml   # Scrape config
│   └── grafana/                    # Auto-provisioned datasource + dashboard
│
├── docs/                           # Architecture docs & implementation blueprint
├── docker-compose.yml              # Full stack (Postgres, Redis, Backend, Prometheus, Grafana)
└── Makefile                        # Dev commands
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| **Backend** | Go 1.22, Chi router, sqlc, gRPC |
| **Database** | PostgreSQL 16 + pgvector, Row-Level Security |
| **Cache** | Redis 7 |
| **Observability** | Prometheus, Grafana, structured JSON logging (slog) |
| **Containerisation** | Docker, Docker Compose |
| **LLM** | Provider-agnostic (Claude, Gemini, etc. via `LLMClient` interface) |

---

## Quick Start

```bash
# 1. Clone and configure
cp backend/.env.example backend/.env

# 2. Start the full stack (Postgres, Redis, Backend, Prometheus, Grafana)
make up-build

# 3. Run database migrations
make migrate

# 4. Verify health
make healthz    # → {"status":"ok","version":"dev"}
make readyz     # → {"status":"ok","checks":{"postgres":"ok"}}

# 5. Run tests
make test
```

---

## Observability Endpoints

| Endpoint | URL | Purpose |
|---|---|---|
| **Liveness** | `http://localhost:8080/healthz` | Always 200 if the process is alive |
| **Readiness** | `http://localhost:8080/readyz` | 200 only when Postgres is reachable |
| **Metrics** | `http://localhost:8080/metrics` | Prometheus scrape target |
| **Prometheus** | `http://localhost:9090` | Query and alert on metrics |
| **Grafana** | `http://localhost:3000` | Pre-built dashboard (admin / meridien) |

### Metrics Tracked

- `meridien_http_requests_total` — HTTP request count (method / path / status)
- `meridien_http_request_duration_seconds` — HTTP latency distribution
- `meridien_agent_turns_total` — Mera conversation turns (channel / outcome)
- `meridien_agent_turn_duration_seconds` — End-to-end agent latency
- `meridien_llm_tokens_total` — LLM token consumption
- `meridien_rag_query_duration_seconds` — pgvector search latency
- `meridien_orders_placed_total` — Orders placed (agent / portal)
- `meridien_order_validation_errors_total` — Validation failures by reason
- `meridien_db_query_duration_seconds` — Database query latency

---

## Makefile Commands

```bash
make up          # Start all containers
make up-build    # Start all containers (rebuild backend)
make down        # Stop all containers
make logs        # Tail backend logs
make migrate     # Run database migrations
make migrate-down # Rollback 1 migration
make sqlc        # Regenerate sqlc code from queries
make test        # Run all tests with race detection + coverage
make vet         # Run go vet
make lint        # Run golangci-lint
make run         # Run backend locally (outside Docker)
make healthz     # Check liveness endpoint
make readyz      # Check readiness endpoint
make metrics     # Preview Prometheus metrics
```

---

## License

Proprietary. Copyright © 2024-2025 MERIDIEN.

**Muhammad Ali** — [GitHub](https://github.com/mu7ammad-3li/) · [LinkedIn](https://linkedin.com/in/muhammad-3lii) · muhammad.3lii@gmail.com
