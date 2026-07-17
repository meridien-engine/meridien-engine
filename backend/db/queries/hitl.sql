-- name: ListHITLTraces :many
SELECT 
  it.id AS trace_id,
  it.interaction_log_id,
  it.retrieved_contexts,
  it.system_prompt,
  it.raw_agent_thoughts,
  it.tools_called,
  it.workflow_id,
  it.hitl_status,
  it.suspended_at,
  it.expires_at,
  it.created_at AS trace_created_at,
  il.id AS log_id,
  il.business_id,
  il.customer_id,
  il.channel,
  il.inbound_msg,
  il.outbound_msg,
  il.tokens_used,
  il.latency_ms,
  il.created_at AS log_created_at
FROM interaction_traces it
JOIN interaction_logs il ON it.interaction_log_id = il.id
WHERE il.business_id = sqlc.arg(business_id)
  AND it.hitl_status != 'none'
ORDER BY it.suspended_at DESC NULLS LAST, il.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: UpdateHITLStatus :exec
UPDATE interaction_traces
SET 
  hitl_status = sqlc.arg(hitl_status)
WHERE interaction_traces.id = sqlc.arg(trace_id)
  AND interaction_traces.interaction_log_id IN (
    SELECT interaction_logs.id FROM interaction_logs WHERE interaction_logs.business_id = sqlc.arg(business_id)
  );
