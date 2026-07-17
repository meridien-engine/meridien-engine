-- ============================================================
-- Migration 004 — Knowledge Base & AI Observability
-- Tables: knowledge_nodes, interaction_logs, interaction_traces
--
-- knowledge_nodes: pgvector-backed document chunks for RAG.
--   RLS ensures merchants only retrieve their own documents.
--
-- interaction_logs: structural metadata about every Mera turn.
-- interaction_traces: deep AI observability — the exact prompt,
--   retrieved chunks, agent thoughts, and tools called.
--   Drives the Compass "AI transparency" dashboard.
-- ============================================================

BEGIN;

-- ── pgvector extension ────────────────────────────────────────────────────────
-- Requires the pgvector Postgres extension to be installed.
-- The docker-compose Postgres image should include it (pgvector/pgvector:pg16).

CREATE EXTENSION IF NOT EXISTS vector;

-- ── Knowledge Nodes ───────────────────────────────────────────────────────────
-- Each row is a single semantic chunk from a merchant-uploaded document
-- (PDF, FAQ, manual). The embedding column holds the vector representation
-- produced by the embedding model (1536 dims = OpenAI text-embedding-3-small).
--
-- Nearest-neighbour retrieval is performed with an IVFFlat index.
-- The index is created after initial data load for efficiency.

CREATE TABLE knowledge_nodes (
  id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id UUID         NOT NULL REFERENCES businesses(id),
  source_name VARCHAR(255) NOT NULL, -- Original document filename / title
  content     TEXT         NOT NULL, -- Raw text of this chunk
  embedding   vector(768)  NOT NULL, -- Embedding vector (768 dims = Gemini text-embedding-004)
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_knowledge_nodes_business_id ON knowledge_nodes(business_id);

-- IVFFlat ANN index — created here with lists=100 as a reasonable default.
-- Re-create with a larger lists value once dataset grows beyond ~1M rows.
CREATE INDEX idx_knowledge_nodes_embedding
  ON knowledge_nodes USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);

ALTER TABLE knowledge_nodes ENABLE ROW LEVEL SECURITY;

CREATE POLICY knowledge_nodes_isolation ON knowledge_nodes
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE knowledge_nodes FORCE ROW LEVEL SECURITY;

-- ── Interaction Logs ──────────────────────────────────────────────────────────
-- One row per complete Mera conversational turn. Lightweight metadata
-- for listing, searching, and aggregating conversations in Compass.

CREATE TABLE interaction_logs (
  id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id  UUID        NOT NULL REFERENCES businesses(id),
  customer_id  UUID        NOT NULL REFERENCES customer_profiles(id),
  channel      VARCHAR(50) NOT NULL, -- 'whatsapp' | 'messenger' | 'web'
  inbound_msg  TEXT        NOT NULL, -- Raw customer message
  outbound_msg TEXT        NOT NULL, -- Mera's final reply
  tokens_used  INT         NOT NULL DEFAULT 0,
  latency_ms   INT         NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT interaction_logs_channel_check CHECK (channel IN ('whatsapp', 'messenger', 'web'))
);

CREATE INDEX idx_interaction_logs_business_id ON interaction_logs(business_id);
CREATE INDEX idx_interaction_logs_customer_id ON interaction_logs(customer_id);
CREATE INDEX idx_interaction_logs_created_at  ON interaction_logs(business_id, created_at DESC);

ALTER TABLE interaction_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY interaction_logs_isolation ON interaction_logs
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE interaction_logs FORCE ROW LEVEL SECURITY;

-- ── Interaction Traces ────────────────────────────────────────────────────────
-- Detailed AI observability record linked 1:1 to an interaction_log.
-- Stored as a separate table so the logs table stays narrow and fast
-- for list queries; traces are fetched on-demand when inspecting a turn.
--
-- retrieved_contexts: JSONB array of { content, score } objects
-- tools_called:       JSONB array of { tool_name, args, result } objects

CREATE TABLE interaction_traces (
  id                 UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  interaction_log_id UUID        NOT NULL REFERENCES interaction_logs(id) ON DELETE CASCADE,
  retrieved_contexts JSONB       NOT NULL DEFAULT '[]',
  system_prompt      TEXT        NOT NULL,
  raw_agent_thoughts TEXT        NOT NULL, -- Full chain-of-thought / scratchpad from LLM
  tools_called       JSONB       NOT NULL DEFAULT '[]',
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_interaction_traces_log_id ON interaction_traces(interaction_log_id);

-- Isolation flows through interaction_logs → business_id.
ALTER TABLE interaction_traces ENABLE ROW LEVEL SECURITY;

CREATE POLICY interaction_traces_isolation ON interaction_traces
  USING (
    EXISTS (
      SELECT 1 FROM interaction_logs
      WHERE interaction_logs.id = interaction_traces.interaction_log_id
    )
  );

ALTER TABLE interaction_traces FORCE ROW LEVEL SECURITY;

COMMIT;
