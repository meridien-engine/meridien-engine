-- Migration 001 — Core Foundation (DOWN)
-- Tears down everything created in the up migration.
-- Drop order is reverse of creation order (foreign key constraints).

BEGIN;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS join_requests;
DROP TABLE IF EXISTS user_business_memberships;
DROP TABLE IF EXISTS businesses;
DROP TABLE IF EXISTS business_categories;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();

COMMIT;
