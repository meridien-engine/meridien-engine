-- name: CreateInteractionLog :one
INSERT INTO interaction_logs (
  business_id, customer_id, channel, inbound_msg, outbound_msg, tokens_used, latency_ms
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: CreateInteractionTrace :one
INSERT INTO interaction_traces (
  interaction_log_id, retrieved_contexts, system_prompt, raw_agent_thoughts, tools_called
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetInteractionWithTrace :one
-- Fetches a single interaction log and its associated trace for Compass.
SELECT
  il.id,
  il.business_id,
  il.customer_id,
  il.channel,
  il.inbound_msg,
  il.outbound_msg,
  il.tokens_used,
  il.latency_ms,
  il.created_at,
  it.retrieved_contexts,
  it.system_prompt,
  it.raw_agent_thoughts,
  it.tools_called
FROM interaction_logs il
LEFT JOIN interaction_traces it ON it.interaction_log_id = il.id
WHERE il.id = $1;

-- name: ListInteractionLogs :many
-- Paginated list for Compass dashboard.
SELECT * FROM interaction_logs
ORDER BY created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: ListInteractionLogsByCustomer :many
SELECT * FROM interaction_logs
WHERE customer_id = $1
ORDER BY created_at DESC;
