-- ============================================================
-- Migration 001 — Core Foundation
-- Tables: users, business_categories, businesses,
--         user_business_memberships
--
-- This is the irreducible core. No branches. No POS.
-- No inventory. No features. Just the entities that
-- everything else will eventually depend on.
-- ============================================================

BEGIN;

-- ── Extensions ───────────────────────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Shared trigger: auto-update updated_at ───────────────────────────────────

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ── Users ─────────────────────────────────────────────────────────────────────
-- Global entities. Not coupled to any business at creation.
-- A user establishes business context after login via a separate token exchange.

CREATE TABLE users (
  id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  email         VARCHAR(255) NOT NULL,
  first_name    VARCHAR(100) NOT NULL,
  last_name     VARCHAR(100) NOT NULL,
  phone         VARCHAR(20),
  password_hash TEXT        NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ,

  CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE INDEX idx_users_email      ON users(email)      WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Business Categories ───────────────────────────────────────────────────────
-- Predefined system list. Seeded once. Not tenant-scoped.

CREATE TABLE business_categories (
  id      UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  name    VARCHAR(100) NOT NULL,
  name_ar VARCHAR(100) NOT NULL,
  slug    VARCHAR(100) NOT NULL,

  CONSTRAINT business_categories_slug_unique UNIQUE (slug)
);

-- Seed: initial category list
INSERT INTO business_categories (name, name_ar, slug) VALUES
  ('Retail',       'تجزئة',         'retail'),
  ('Food & Drink', 'طعام وشراب',     'food-drink'),
  ('Electronics',  'إلكترونيات',     'electronics'),
  ('Fashion',      'أزياء',          'fashion'),
  ('Healthcare',   'رعاية صحية',     'healthcare'),
  ('Services',     'خدمات',          'services'),
  ('Education',    'تعليم',          'education'),
  ('Other',        'أخرى',           'other');

-- ── Businesses ────────────────────────────────────────────────────────────────
-- The multi-tenant root. Every piece of business data is scoped to a business_id.
-- RLS is enabled on this table and all downstream tables.

CREATE TABLE businesses (
  id               UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
  name             VARCHAR(255) NOT NULL,
  slug             VARCHAR(100) NOT NULL,       -- immutable after creation
  owner_id         UUID         NOT NULL REFERENCES users(id),
  category_id      UUID         REFERENCES business_categories(id),
  contact_phone    VARCHAR(20),
  contact_email    VARCHAR(255),
  logo_url         TEXT,
  plan             VARCHAR(50)  NOT NULL DEFAULT 'free',   -- label only, no enforcement yet
  status           VARCHAR(20)  NOT NULL DEFAULT 'active',
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at       TIMESTAMPTZ,

  CONSTRAINT businesses_slug_unique        UNIQUE (slug),
  CONSTRAINT businesses_status_check       CHECK  (status IN ('active', 'suspended', 'trial')),
  CONSTRAINT businesses_plan_check         CHECK  (plan   IN ('free', 'starter', 'pro'))
);

CREATE INDEX idx_businesses_owner_id   ON businesses(owner_id)   WHERE deleted_at IS NULL;
CREATE INDEX idx_businesses_slug       ON businesses(slug)        WHERE deleted_at IS NULL;
CREATE INDEX idx_businesses_deleted_at ON businesses(deleted_at)  WHERE deleted_at IS NOT NULL;

CREATE TRIGGER businesses_updated_at
  BEFORE UPDATE ON businesses
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Row-Level Security: businesses ───────────────────────────────────────────
-- businesses itself is not RLS-gated (users need to find businesses to join them).
-- All downstream tables (memberships, products, orders, etc.) are RLS-gated.

-- ── User Business Memberships ─────────────────────────────────────────────────
-- The join table between users and businesses. Carries the user's role
-- within that business. A user may be a member of multiple businesses
-- with different roles in each.

CREATE TABLE user_business_memberships (
  id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID        NOT NULL REFERENCES users(id),
  business_id UUID        NOT NULL REFERENCES businesses(id),
  role        VARCHAR(20) NOT NULL,
  status      VARCHAR(20) NOT NULL DEFAULT 'active',
  invited_by  UUID        REFERENCES users(id),   -- NULL if owner or self-joined
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT memberships_user_business_unique UNIQUE (user_id, business_id),
  CONSTRAINT memberships_role_check   CHECK (role   IN ('owner', 'admin', 'manager', 'viewer')),
  CONSTRAINT memberships_status_check CHECK (status IN ('active', 'suspended'))
);

CREATE INDEX idx_memberships_user_id     ON user_business_memberships(user_id);
CREATE INDEX idx_memberships_business_id ON user_business_memberships(business_id);

CREATE TRIGGER memberships_updated_at
  BEFORE UPDATE ON user_business_memberships
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- RLS on memberships
ALTER TABLE user_business_memberships ENABLE ROW LEVEL SECURITY;

CREATE POLICY memberships_business_isolation ON user_business_memberships
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE user_business_memberships FORCE ROW LEVEL SECURITY;

-- ── Join Requests ─────────────────────────────────────────────────────────────
-- User-initiated: a user finds a business by slug and requests to join.

CREATE TABLE join_requests (
  id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID        NOT NULL REFERENCES users(id),
  business_id UUID        NOT NULL REFERENCES businesses(id),
  message     TEXT,
  role        VARCHAR(20) NOT NULL DEFAULT 'viewer',
  status      VARCHAR(20) NOT NULL DEFAULT 'pending',
  reviewed_by UUID        REFERENCES users(id),
  reviewed_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT join_requests_status_check CHECK (status IN ('pending', 'approved', 'rejected')),
  CONSTRAINT join_requests_role_check   CHECK (role   IN ('admin', 'manager', 'viewer'))
);

CREATE INDEX idx_join_requests_business_id ON join_requests(business_id);
CREATE INDEX idx_join_requests_user_id     ON join_requests(user_id);
CREATE INDEX idx_join_requests_status      ON join_requests(status) WHERE status = 'pending';

ALTER TABLE join_requests ENABLE ROW LEVEL SECURITY;

CREATE POLICY join_requests_business_isolation ON join_requests
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE join_requests FORCE ROW LEVEL SECURITY;

-- ── Invitations ───────────────────────────────────────────────────────────────
-- Admin/owner-initiated: invite a user to the business by email.

CREATE TABLE invitations (
  id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id UUID        NOT NULL REFERENCES businesses(id),
  email       VARCHAR(255) NOT NULL,
  role        VARCHAR(20) NOT NULL,
  token       VARCHAR(255) NOT NULL,             -- secure random token
  invited_by  UUID        NOT NULL REFERENCES users(id),
  status      VARCHAR(20) NOT NULL DEFAULT 'pending',
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT invitations_token_unique  UNIQUE (token),
  CONSTRAINT invitations_status_check  CHECK  (status IN ('pending', 'accepted', 'expired')),
  CONSTRAINT invitations_role_check    CHECK  (role   IN ('admin', 'manager', 'viewer'))
);

CREATE INDEX idx_invitations_business_id ON invitations(business_id);
CREATE INDEX idx_invitations_token       ON invitations(token);
CREATE INDEX idx_invitations_email       ON invitations(email);

ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;

CREATE POLICY invitations_business_isolation ON invitations
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE invitations FORCE ROW LEVEL SECURITY;

-- ── Audit Log ─────────────────────────────────────────────────────────────────
-- Immutable append-only trail. No updates, no deletes, ever.
-- Every significant write in the system drops a row here.

CREATE TABLE audit_logs (
  id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id UUID        NOT NULL REFERENCES businesses(id),
  user_id     UUID        REFERENCES users(id),
  action      VARCHAR(100) NOT NULL,   -- e.g. 'membership.created', 'order.status_changed'
  entity_type VARCHAR(100) NOT NULL,   -- e.g. 'membership', 'order'
  entity_id   UUID,
  payload     JSONB,                   -- before/after state or relevant context
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_business_id  ON audit_logs(business_id);
CREATE INDEX idx_audit_logs_entity       ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created_at   ON audit_logs(created_at DESC);

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY audit_logs_business_isolation ON audit_logs
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;

COMMIT;
