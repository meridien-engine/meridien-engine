# Future Roadmap & TODOs

## Cross-Tenant AI Federation (Global UCM)
**Goal:** Implement a premium subscription tier ("SS enhancements") that allows opted-in merchants to securely pool and share anonymized customer interaction data to enhance the Synapse AI brain.

### Context
Currently, the `CustomerProfile` and its AI-generated `SemanticSummary` are strictly isolated per merchant using PostgreSQL Row-Level Security (`business_id`). Buyers (consumers) do not have global accounts; only merchants do.

### Implementation Strategy
- **Global UCM (Unified Customer Model):** Create a global data layer that aggregates behavioral data and successful selling techniques across the platform based on matching contact information in the `customer_channels` table (e.g., matching a WhatsApp phone number).
- **Business-Level Scoping:** The `SemanticSummary` field remains a strictly business-level value so that merchants retain their own private context notes on the buyer.
- **Opt-In Access:** Only merchants subscribed to the enhanced program will have access to the global UCM insights.
- **Data Pipeline:** Build a background process to merge successful behaviors and contextual info from isolated tenant interactions into the global UCM without violating B2B privacy boundaries for non-participating tenants.
