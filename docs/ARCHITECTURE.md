# Meridien Engine — Architecture

**Version:** 1.0 (Phase 1 — ERP Core)
**Last updated:** May 2026

This document is the source of truth for architectural decisions.
Every significant decision is recorded here before code is written.
If something is built that isn't in this doc, the doc is wrong — update it.

---

## System Overview

Meridien Engine is a unified backend platform combining a multi-tenant ERP/CRM core
with an AI reasoning layer. The AI observes and suggests; it never writes to business
data directly. Humans stay in the loop on exceptions, not every message.

```
Phase 1 (now):   Clean ERP/CRM backend. Multi-tenant. Tested.
Phase 2 (next):  AI intelligence layer — Synapse, Knowledge, Brain, Compass.
Phase 3:         Operator surface — Compass and Meridien React frontends.
Phase 4:         Mera live on social channels.
Phase 5:         Scale features (branches, advanced analytics, fine-tuning).
```

---

## Architectural Decisions (Locked)

| Decision | Choice | Reason |
|---|---|---|
| Architecture style | Modular monolith | Solo engineer, no production traffic yet. Extract services when there is a measured reason. |
| Backend language | Go | Concurrency, performance, type safety. |
| ORM | sqlc | Type-safe Go generated from raw SQL. No magic. Full control over queries. |
| HTTP framework | Gin | Lightweight, fast, production-ready. |
| Database | PostgreSQL 15+ | RLS for multi-tenancy, pgvector for embeddings later. One DB. |
| Multi-tenancy | PostgreSQL Row-Level Security | Enforced at the database layer, not just code. |
| Cache | Redis 7+ | Token blacklist, rate limiting, event queues later. |
| Auth | Two-token JWT (generic → scoped) | Proven in MERIDIEN v1. Users are global; business context is session-scoped. |
| Frontend | React + TypeScript | Three apps: meridien, compass, mera. |
| LLM provider | Claude API | Reasoning quality, prompt caching, structured output. |
| Migrations | golang-migrate | SQL-first, explicit, reversible. |
| Test strategy | Table-driven, 60% coverage floor, CI blocks merge below threshold. |
| Repo structure | Monorepo | Shared types, single deploy pipeline, no version mismatch. Split when deployment cadences diverge. |

---

## What Is NOT Being Built in Phase 1

- POS (point of sale)
- Branches / multi-location inventory
- Any AI or LLM features
- Frontend (backend-first)

These are not cancelled — they are deferred until the foundation is solid and
a real customer use case demands them.

---

## Repository Structure

```
meridien-engine/
├── backend/
│   ├── cmd/server/main.go          # Entry point + DI wiring
│   ├── internal/
│   │   ├── auth/                   # Auth handlers, services, JWT logic
│   │   ├── business/               # Business, membership, join, invite
│   │   ├── customer/               # Customer CRM
│   │   ├── product/                # Product catalog
│   │   ├── order/                  # Orders, line items, payments
│   │   ├── common/                 # Shared domain logic
│   │   └── db/                     # sqlc-generated code (do not edit manually)
│   ├── db/
│   │   ├── migrations/             # SQL migration files (golang-migrate)
│   │   └── queries/                # SQL query definitions (sqlc input)
│   ├── pkg/
│   │   ├── middleware/             # Auth middleware, rate limiting
│   │   ├── response/               # Standard response envelopes
│   │   └── validate/               # Input validation helpers
│   ├── sqlc.yaml
│   ├── go.mod
│   └── .env.example
│
├── frontend/
│   ├── apps/
│   │   ├── meridien/               # ERP operator interface (React + TypeScript)
│   │   ├── compass/                # AI control panel + analytics (React + TypeScript)
│   │   └── mera/                   # Customer-facing AI agent UI (React + TypeScript)
│   └── packages/shared/            # Shared types, hooks, UI components
│
├── docs/                           # Architecture, ADRs, API contracts
├── scripts/                        # Dev tooling, seed scripts
├── .github/workflows/ci.yml        # CI pipeline
├── docker-compose.yml
├── Makefile
└── .gitignore
```

---

## Database Schema (Phase 1)

### Migration History

| Migration | Tables | Status |
|---|---|---|
| 000001_core_foundation | users, business_categories, businesses, user_business_memberships, join_requests, invitations, audit_logs | ✅ |

### Core Tables

```
users                       — global, no business_id
businesses                  — multi-tenant root
business_categories         — predefined system list (seeded)
user_business_memberships   — user ↔ business with role
join_requests               — user-initiated business joins
invitations                 — admin/owner-initiated invites
audit_logs                  — immutable append-only trail
```

Migrations 000002+ will add: customers, products, orders.

### RLS Pattern

Every repository operation that reads or writes business-scoped data wraps in a
transaction that sets the business context:

```go
func businessTx(ctx context.Context, db *pgxpool.Pool, businessID uuid.UUID, fn func(pgx.Tx) error) error {
    tx, err := db.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    if _, err := tx.Exec(ctx, "SET LOCAL app.current_business = $1", businessID); err != nil {
        return err
    }
    if err := fn(tx); err != nil { return err }
    return tx.Commit(ctx)
}
```

PostgreSQL RLS policies use `current_setting('app.current_business', true)` to enforce
row-level isolation. The `true` parameter means it returns NULL (not an error) if the
setting is not set — important for operations that run outside business context (user
registration, business listing).

---

## Authentication Flow

```
1. POST /auth/register       → create user (no business attached)
2. POST /auth/login          → return generic JWT { user_id, type="generic" }
3. GET  /auth/businesses     → list businesses the user is a member of
4. POST /auth/use-business/:id
                             → validate active membership
                             → return scoped JWT { user_id, business_id, role, type="scoped" }
5. All business API calls    → require scoped JWT
6. Switch business           → repeat from step 4
```

### JWT Token Types

| Field | Generic | Scoped |
|---|---|---|
| type | "generic" | "scoped" |
| user_id | ✅ | ✅ |
| business_id | ❌ | ✅ |
| role | ❌ | ✅ |
| jti | ✅ | ✅ |

### Token Revocation

JTI stored in Redis with TTL equal to token expiry on logout.
Redis is fail-open — the app runs normally if Redis is unavailable,
accepting the minor security tradeoff for operational stability.

---

## API Conventions

### Response Envelopes

```json
// Success
{ "success": true, "message": "...", "data": {} }

// Error
{ "success": false, "error": "SNAKE_CASE_CODE", "message": "Human-readable description" }

// Paginated
{
  "success": true,
  "data": [...],
  "meta": { "total": 100, "page": 1, "per_page": 20, "total_pages": 5 }
}
```

### REST Conventions

```
GET    /api/v1/{resource}       — list (paginated)
GET    /api/v1/{resource}/:id   — get by ID
POST   /api/v1/{resource}       — create
PUT    /api/v1/{resource}/:id   — full update
PATCH  /api/v1/{resource}/:id   — partial update
DELETE /api/v1/{resource}/:id   — soft delete (sets deleted_at)
```

### Layering

```
Handler  — parse request, call service, format response. No business logic.
Service  — business logic, orchestration, validation. No SQL.
DB (sqlc) — type-safe generated queries. No business logic.
```

---

## Role Set

| Role | Capabilities |
|---|---|
| owner | Everything. Immutable. Auto-assigned at business creation. |
| admin | Approve joins, invite users, manage all business data. |
| manager | Manage products, orders, customers. |
| viewer | Read-only. |

---

## Security

| Concern | Implementation |
|---|---|
| Passwords | bcrypt, cost 12 |
| JWT | HS256, 24h expiry, unique JTI per token |
| Token revocation | JTI in Redis with TTL on logout |
| Rate limiting | Fixed-window per IP on auth endpoints |
| Data isolation | PostgreSQL RLS + SET LOCAL per transaction |
| SQL injection | sqlc prepared statements — no raw string interpolation |
| Soft deletes | deleted_at on all business entities |

---

## What the Old Codebase (MERIDIEN v1) Taught Us

| Lesson | What We're Doing Differently |
|---|---|
| POS and branches were built before any customer needed them | Phase 1 has neither. They come back when a real use case demands them. |
| GORM hides SQL and makes complex queries awkward | sqlc: write the SQL you know, get clean Go back. |
| No test coverage is serious debt | 60% coverage floor enforced by CI from day one. |
| Flutter was the wrong frontend choice for a B2B SaaS | React + TypeScript across all three frontends. |
| Architecture doc lagged behind the code | This doc is written before code. It is the source of truth. |
| No CI until late | GitHub Actions CI configured in Phase 1, never turned off. |
