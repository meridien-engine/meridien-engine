-- name: ListTraces :many
SELECT 
  it.id AS trace_id,
  it.interaction_log_id,
  it.retrieved_contexts,
  it.system_prompt,
  it.raw_agent_thoughts,
  it.tools_called,
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
ORDER BY il.created_at DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);
