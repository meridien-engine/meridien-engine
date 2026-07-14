-- ============================================================
-- Migration 005 — HITL Suspension Tracking
-- Adds workflow suspension state to interaction_traces so the
-- background expiry checker can find and escalate stalled HITL reviews.
-- ============================================================

BEGIN;

ALTER TABLE interaction_traces
  ADD COLUMN workflow_id  VARCHAR(128),
  ADD COLUMN hitl_status  VARCHAR(50)  NOT NULL DEFAULT 'none',
  ADD COLUMN suspended_at TIMESTAMPTZ,
  ADD COLUMN expires_at   TIMESTAMPTZ;

-- Valid hitl_status values: 'none' | 'pending' | 'approved' | 'rejected' | 'timed_out'
ALTER TABLE interaction_traces
  ADD CONSTRAINT interaction_traces_hitl_status_check
  CHECK (hitl_status IN ('none', 'pending', 'approved', 'rejected', 'timed_out'));

-- Partial index: fast lookup for the background expiry checker.
CREATE INDEX idx_interaction_traces_pending_hitl
  ON interaction_traces (expires_at)
  WHERE hitl_status = 'pending';

COMMIT;
