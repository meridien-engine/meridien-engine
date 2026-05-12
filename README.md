# Meridien Engine
Meridien Engine is a unified platform that powers a bussiness intelligence and cusstomer ecosystem - combining a multi-tenant ERP/CRM core with AI reasoning layer that lets operators undrestand their customers deeply and serve them intelligently, without givinig the AI unsupervised acess to bussiness data. 

---
## Prior Work

This project is a ground-up rewrite of [MERIDIEN](https://github.com/meridien-engine/MERIDIEN),
the original ERP/CRM system that informed this architecture.
The old repository serves as a reference for decisions made and lessons learned —
not as a codebase to migrate from.

---

## Repository Structure

theis is a <u>proposed</u> refined project structure 

```
meridien-engine/
├── backend/                    # Go backend (modular monolith)
│   ├── cmd/server/             # Entry point + dependency injection
│   ├── internal/               # Domain packages
│   │   ├── auth/               # Authentication, JWT, token management
│   │   ├── business/           # Businesses, memberships, invitations, join requests
│   │   ├── customer/           # Customer CRM
│   │   ├── product/            # Product catalog, categories
│   │   ├── order/              # Orders, line items, payments
│   │   └── common/             # Shared domain logic
│   ├── db/
│   │   ├── migrations/         # golang-migrate SQL migrations
│   │   └── queries/            # sqlc SQL query definitions
│   └── pkg/                    # Cross-cutting infrastructure
│       ├── middleware/          # Auth, rate limiting
│       ├── response/           # Response envelopes
│       └── validate/           # Input validation helpers
│
├── frontend/                   # React + TypeScript (three apps)
│   ├── apps/
│   │   ├── meridien/           # ERP operator interface
│   │   ├── compass/            # AI control panel + analytics
│   │   └── mera/               # Customer-facing AI agent UI
│   └── packages/shared/        # Shared types, hooks, components
│
├── docs/                       # Architecture, ADRs, API contracts
└── scripts/                    # Dev tooling, migration runner, seed scripts
```

## Tech Stack

| Layer | Technology |
|---|---|
| **Backend** | Go, Gin, sqlc |
| **Database** | PostgreSQL 15+ with Row-Level Security, pgvector |
| **Cache**| Redis 7+ |
| **Frontend** | React + TypeScript |
| **Auth** | JWT HS256, two-token pattern |
| **LLM** | Still under review  |
