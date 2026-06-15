-- Migration 003 — Synapse: Customer Intelligence (DOWN)

BEGIN;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_customer_id_fk;

DROP TABLE IF EXISTS customer_channels;
DROP TABLE IF EXISTS customer_profiles;

COMMIT;
