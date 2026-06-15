-- Migration 004 — Knowledge Base & AI Observability (DOWN)

BEGIN;

DROP TABLE IF EXISTS interaction_traces;
DROP TABLE IF EXISTS interaction_logs;
DROP TABLE IF EXISTS knowledge_nodes;

DROP EXTENSION IF EXISTS vector;

COMMIT;
