-- ============================================================
-- Migration 003 — Synapse: Customer Intelligence
-- Tables: customer_profiles, customer_channels
--
-- customer_profiles is the Unified Customer Model (UCM).
-- A single customer can be reached across WhatsApp, Messenger,
-- and Web — all merged into one profile per business.
--
-- After this migration we also backfill the FK constraint on
-- orders.customer_id which was deferred until now.
-- ============================================================

BEGIN;

-- ── Customer Profiles (UCM) ───────────────────────────────────────────────────
-- One row per unique customer per business. The semantic_summary field
-- is overwritten periodically by the Synapse background summarizer as
-- new interaction data accumulates.

CREATE TABLE customer_profiles (
  id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id      UUID        NOT NULL REFERENCES businesses(id),
  unified_name     VARCHAR(255),
  customer_tier    VARCHAR(50) NOT NULL DEFAULT 'standard',
  semantic_summary TEXT,        -- LLM-generated rolling summary. May be NULL initially.
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT customer_tier_check CHECK (customer_tier IN ('standard', 'silver', 'gold'))
);

CREATE INDEX idx_customer_profiles_business_id ON customer_profiles(business_id);

CREATE TRIGGER customer_profiles_updated_at
  BEFORE UPDATE ON customer_profiles
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE customer_profiles ENABLE ROW LEVEL SECURITY;

CREATE POLICY customer_profiles_isolation ON customer_profiles
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE customer_profiles FORCE ROW LEVEL SECURITY;

-- ── Customer Channels ─────────────────────────────────────────────────────────
-- Maps external platform identifiers to a unified customer profile.
-- Example: WhatsApp "+9665XXXXXXXX" → customer_profile.id
-- Synapse uses this lookup on every inbound message to resolve or
-- create the correct UCM record.

CREATE TABLE customer_channels (
  id                  UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  customer_profile_id UUID        NOT NULL REFERENCES customer_profiles(id) ON DELETE CASCADE,
  channel_type        VARCHAR(50) NOT NULL, -- 'whatsapp' | 'messenger' | 'web'
  channel_external_id VARCHAR(255) NOT NULL, -- Phone number or platform user ID
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT customer_channels_unique UNIQUE (customer_profile_id, channel_type, channel_external_id),
  CONSTRAINT customer_channels_type_check CHECK (channel_type IN ('whatsapp', 'messenger', 'web'))
);

CREATE INDEX idx_customer_channels_lookup
  ON customer_channels(channel_type, channel_external_id);

-- customer_channels isolation flows through customer_profile → business_id.
ALTER TABLE customer_channels ENABLE ROW LEVEL SECURITY;

CREATE POLICY customer_channels_isolation ON customer_channels
  USING (
    EXISTS (
      SELECT 1 FROM customer_profiles
      WHERE customer_profiles.id = customer_channels.customer_profile_id
    )
  );

ALTER TABLE customer_channels FORCE ROW LEVEL SECURITY;

-- ── Backfill FK: orders.customer_id → customer_profiles(id) ──────────────────
-- orders was created in migration 002 with a plain UUID column.
-- Now that customer_profiles exists, we add the FK constraint.

ALTER TABLE orders
  ADD CONSTRAINT orders_customer_id_fk
    FOREIGN KEY (customer_id) REFERENCES customer_profiles(id);

COMMIT;
