# System Flow Scenarios & Runtime Logic

This document details the primary runtime scenarios, request flows, and architectural invariants of the **Meridien Engine** modular monolith. It provides sequence and logic flows using Mermaid diagrams to illustrate how multi-tenancy, transactional safety, and contextual tracing are enforced across the services.

---

## 1. Multi-Tenant Context Propagation & Request Lifecycle

To guarantee absolute isolation between merchants, tenant identification is extracted at the entry points (HTTP/gRPC) and propagated through the application layers down to the database session context.

### Sequence Flow

```mermaid
sequenceDiagram
    autonumber
    actor Client
    participant Rest as HTTP REST Middleware / gRPC Interceptor
    participant Service as Domain Service (ERP / Synapse)
    participant Repo as Repository Helper
    participant DB as PostgreSQL Database

    Client->>Rest: Request with Header/Metadata (X-Business-ID: <UUID>)
    Note over Rest: Parse UUID & inject into Go context<br/>via WithBusinessID(ctx, businessID)
    Rest->>Service: Call Service Method (ctx, args...)
    Service->>Repo: Call Repository Query (ctx, queryParams)
    Repo->>Repo: Retrieve businessID = BusinessIDFromContext(ctx)
    Repo->>DB: ExecWithTenant(ctx, db, businessID, fn)
    Note over DB: Start Transaction (BEGIN)
    Repo->>DB: SET LOCAL app.current_business = businessID
    Repo->>DB: Execute sqlc queries (SELECT / INSERT / UPDATE)
    Note over DB: Database evaluates RLS policies:<br/>business_id = current_setting('app.current_business')::uuid
    DB-->>Repo: Query Results
    Note over DB: End Transaction (COMMIT)
    Repo-->>Service: Return domain entities / error
    Service-->>Rest: Return response payload
    Rest-->>Client: JSON Response / gRPC Stream
```

### Key Components:
* **Context Setter/Getter**: [tenant.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/tenant.go) implements [WithBusinessID](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/tenant.go#L16) and [BusinessIDFromContext](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/tenant.go#L23).
* **Database Session Binding**: [ExecWithTenant](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/tenant.go#L43) wraps transactions, explicitly invoking `SET LOCAL app.current_business` to activate database Row-Level Security (RLS).

---

## 2. Order Placement & Integrity Verification Flow (ERP)

All orders (whether placed via the Merchant Portal or generated autonomously by the AI agent) must pass through the **ERP Order Verification Layer** to enforce inventory constraints and flag pricing anomalies.

### Flow Logic

```mermaid
flowchart TD
    Start([Incoming PlaceNewOrder Request]) --> Auth[Extract BusinessID & Tenant Context]
    Auth --> Lock[Start Tenant Transaction]
    Lock --> CheckStock{1. Iterate Items: Stock Availability?}
    
    CheckStock -- No / Out of Stock --> FailStock[Abort: Return Out-of-Stock Error]
    CheckStock -- Yes --> ComparePrice{2. Submitted Price == Catalog Price?}

    ComparePrice -- Match --> DecrStock[3. Decrement Stock Quantity]
    DecrStock --> SaveOrder[4. Create Order & Items in DB with Status = 'pending']
    SaveOrder --> Commit[5. Commit Transaction]
    Commit --> Done([Order Placed Successfully])

    ComparePrice -- Discrepancy / Mismatch --> FlagReview[3. Decrement Stock Quantity]
    FlagReview --> SaveOrderFlagged[4. Create Order & Items with Status = 'pending_review']
    SaveOrderFlagged --> Commit
    Commit --> ReviewDone([Order Created, Flagged for Merchant Manual Review])

    FailStock --> Rollback[Rollback Transaction]
    Rollback --> EndErr([Return Out of Stock Status / Code])
    
    style FailStock fill:#f9d2d2,stroke:#d9534f,stroke-width:2px;
    style ReviewDone fill:#fff3cd,stroke:#f0ad4e,stroke-width:2px;
    style Done fill:#d4edda,stroke:#5cb85c,stroke-width:2px;
```

### Key Components:
* **gRPC Endpoint**: [PlaceNewOrder](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/grpchandler/order_handler.go#L33) in [order_handler.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/grpchandler/order_handler.go).
* **Core Logic**: [PlaceOrder](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/erp/service.go#L33) in [service.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/erp/service.go).
* **Inventory Control**: [DecrementStock](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/product.go#L74) in [product.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/product.go).

---

## 3. Customer Interaction Log & Trace Flow (Synapse)

When the autonomous agent interacts with a customer, the conversational turns and underlying agent thoughts (including retrieved RAG contexts and tool execution histories) are durably recorded in the Synapse database schema.

### Sequence Flow

```mermaid
sequenceDiagram
    autonumber
    actor Customer
    participant Agent as Autonomous Agent / Mera Gateway
    participant Synapse as Synapse Service
    participant CustomerRepo as Customer Repository
    participant InteractionRepo as Interaction Repository
    participant DB as PostgreSQL Database

    Customer->>Agent: Send message (e.g. WhatsApp)
    Note over Agent: Retrieve RAG context & invoke LLM
    Agent->>Synapse: RecordTurn(ctx, Log, Trace)
    
    Note over Synapse: Check customer registration
    Synapse->>CustomerRepo: GetOrCreateByChannel(ctx, channelDetails)
    CustomerRepo-->>Synapse: Customer Profile ID
    
    Synapse->>InteractionRepo: RecordTurn(ctx, logEntity, traceEntity)
    InteractionRepo->>DB: ExecWithTenant(ctx, db, businessID, tx)
    Note over DB: Start Transaction (BEGIN)
    InteractionRepo->>DB: CreateInteractionLog(...)
    DB-->>InteractionRepo: Log ID & CreatedAt
    InteractionRepo->>DB: CreateInteractionTrace(...) [Includes JSONB data]
    Note over DB: Commit Transaction (COMMIT)
    
    DB-->>InteractionRepo: Complete
    InteractionRepo-->>Synapse: Return Success
    Synapse-->>Agent: Handshake Complete
```

### Key Components:
* **Domain Service**: [RecordTurn](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/synapse/service.go#L41) in [service.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/synapse/service.go).
* **Repository Storage**: [RecordTurn](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/interaction.go#L27) in [interaction.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/interaction.go).

---

## 4. Vector Search & RAG Query Flow (Knowledge)

To keep conversational search latency minimal and decoupled from database lock states, Knowledge base search operations query vector embeddings directly using pgvector similarity metrics.

### Flow Logic

```mermaid
flowchart LR
    Query([Query String]) --> Embed[Generate Vector Embedding]
    Embed --> RetrieveTenant[Extract Tenant context app.current_business]
    RetrieveTenant --> DBQuery[Execute Cosine Distance Vector Search]
    
    subgraph "PostgreSQL pgvector Engine"
        DBQuery --> RLS[Apply RLS policy]
        RLS --> KNN[Select Nearest Neighbors: embedding <=> input_vector]
    end
    
    KNN --> Filter[Filter Results by Similarity Threshold]
    Filter --> Contexts([Return Formatted retrieved_contexts Context Block])
```

### Key Components:
* **gRPC Endpoint**: [QueryKnowledge](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/grpchandler/knowledge_handler.go#L32) in [knowledge_handler.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/grpchandler/knowledge_handler.go).
* **pgvector Queries**: [Query](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/knowledge.go#L27) in [knowledge.go](file:///media/muhammad/FS/2026/meridien-engine/backend/internal/repository/knowledge.go).
