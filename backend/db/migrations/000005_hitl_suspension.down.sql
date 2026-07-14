BEGIN;
DROP INDEX IF EXISTS idx_interaction_traces_pending_hitl;
ALTER TABLE interaction_traces
  DROP CONSTRAINT IF EXISTS interaction_traces_hitl_status_check,
  DROP COLUMN IF EXISTS workflow_id,
  DROP COLUMN IF EXISTS hitl_status,
  DROP COLUMN IF EXISTS suspended_at,
  DROP COLUMN IF EXISTS expires_at;
COMMIT;
