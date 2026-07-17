-- ============================================================
-- Migration 006 — System Secrets Vault
-- Stores API keys and tokens in the database instead of on disk.
-- Keys are scoped per-business (multi-tenant) and encrypted at
-- the application layer before INSERT.
-- ============================================================

BEGIN;

CREATE TABLE system_secrets (
  id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id   UUID         NOT NULL REFERENCES businesses(id),
  key_name      VARCHAR(100) NOT NULL,   -- e.g. 'gemini_api_key', 'meta_page_access_token'
  encrypted_val TEXT         NOT NULL,   -- AES-256-GCM encrypted value (base64-encoded)
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

  -- Each business can only have one value per key_name
  CONSTRAINT system_secrets_biz_key UNIQUE (business_id, key_name)
);

-- RLS: each business can only read its own secrets
ALTER TABLE system_secrets ENABLE ROW LEVEL SECURITY;

CREATE POLICY system_secrets_tenant_isolation ON system_secrets
  USING (business_id = current_setting('app.current_business', true)::uuid);

CREATE TRIGGER system_secrets_updated_at
  BEFORE UPDATE ON system_secrets
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
