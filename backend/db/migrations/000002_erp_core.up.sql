-- ============================================================
-- Migration 002 — ERP Core: Products & Orders
-- Tables: products, orders, order_items
--
-- Single-location inventory model. Stock is tracked directly
-- on the products table (no branch tables).
--
-- Price guardrail: unit_price on order_items is always resolved
-- from the catalog at checkout time. The AI agent submits only
-- SKU + quantity. The ERP service writes the authoritative price.
-- ============================================================

BEGIN;

-- ── Products ──────────────────────────────────────────────────────────────────
-- The merchant's product catalog. Each product belongs to one business.
-- stock_qty is a simple integer counter — decremented atomically on order
-- placement to prevent double-selling.

CREATE TABLE products (
  id          UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id UUID           NOT NULL REFERENCES businesses(id),
  sku         VARCHAR(100)   NOT NULL,
  name        VARCHAR(255)   NOT NULL,
  description TEXT,
  price       NUMERIC(12, 2) NOT NULL,
  stock_qty   INT            NOT NULL DEFAULT 0,
  is_active   BOOLEAN        NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMPTZ,

  CONSTRAINT products_sku_business_unique UNIQUE (business_id, sku),
  CONSTRAINT products_price_non_negative  CHECK  (price >= 0),
  CONSTRAINT products_stock_non_negative  CHECK  (stock_qty >= 0)
);

CREATE INDEX idx_products_business_id ON products(business_id)   WHERE deleted_at IS NULL;
CREATE INDEX idx_products_sku         ON products(business_id, sku) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_deleted_at  ON products(deleted_at)    WHERE deleted_at IS NOT NULL;

CREATE TRIGGER products_updated_at
  BEFORE UPDATE ON products
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE products ENABLE ROW LEVEL SECURITY;

CREATE POLICY products_business_isolation ON products
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE products FORCE ROW LEVEL SECURITY;

-- ── Orders ────────────────────────────────────────────────────────────────────
-- An order belongs to a business (tenant) and links to a customer profile (UCM).
-- total_price is computed and sealed at placement time by the ERP service.
-- The 'source' column distinguishes agent-placed orders from portal-placed orders.

CREATE TABLE orders (
  id            UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
  business_id   UUID           NOT NULL REFERENCES businesses(id),
  customer_id   UUID           NOT NULL, -- FK to customer_profiles(id) — added in migration 003
  total_price   NUMERIC(12, 2) NOT NULL,
  status        VARCHAR(50)    NOT NULL DEFAULT 'pending',
  source        VARCHAR(30)    NOT NULL DEFAULT 'agent', -- 'agent' | 'portal'
  notes         TEXT,
  created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

  CONSTRAINT orders_status_check CHECK (status IN ('pending', 'completed', 'cancelled')),
  CONSTRAINT orders_source_check CHECK (source IN ('agent', 'portal'))
);

CREATE INDEX idx_orders_business_id  ON orders(business_id);
CREATE INDEX idx_orders_customer_id  ON orders(customer_id);
CREATE INDEX idx_orders_status       ON orders(business_id, status);
CREATE INDEX idx_orders_created_at   ON orders(business_id, created_at DESC);

CREATE TRIGGER orders_updated_at
  BEFORE UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

CREATE POLICY orders_business_isolation ON orders
  USING (business_id = current_setting('app.current_business', true)::uuid);

ALTER TABLE orders FORCE ROW LEVEL SECURITY;

-- ── Order Items ───────────────────────────────────────────────────────────────
-- Line items for an order. unit_price is the catalog price at the moment of
-- order placement — sealed here for historical accuracy even if catalog changes.
-- The AI agent never submits a price; price is always resolved from products.sku.

CREATE TABLE order_items (
  id         UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
  order_id   UUID           NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id UUID           NOT NULL REFERENCES products(id),
  sku        VARCHAR(100)   NOT NULL,
  name       VARCHAR(255)   NOT NULL, -- Snapshotted at order time
  quantity   INT            NOT NULL,
  unit_price NUMERIC(12, 2) NOT NULL, -- Catalog price resolved by ERP service

  CONSTRAINT order_items_quantity_positive CHECK (quantity > 0),
  CONSTRAINT order_items_price_non_negative CHECK (unit_price >= 0)
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- order_items inherits tenant isolation through order_id → orders.business_id.
-- We enforce it via a security-safe subquery policy.
ALTER TABLE order_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY order_items_isolation ON order_items
  USING (
    EXISTS (
      SELECT 1 FROM orders
      WHERE orders.id = order_items.order_id
    )
  );

ALTER TABLE order_items FORCE ROW LEVEL SECURITY;

COMMIT;
