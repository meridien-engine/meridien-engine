-- Migration 002 — ERP Core (DOWN)
-- Tears down products, orders, order_items in reverse dependency order.

BEGIN;

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;

COMMIT;
