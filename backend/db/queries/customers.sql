-- name: CreateCustomerProfile :one
INSERT INTO customer_profiles (
  business_id, unified_name, customer_tier
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetCustomerProfileByID :one
SELECT * FROM customer_profiles
WHERE id = $1
LIMIT 1;

-- name: UpdateSemanticSummary :one
UPDATE customer_profiles
SET semantic_summary = $2
WHERE id = $1
RETURNING *;

-- name: UpdateCustomerTier :one
UPDATE customer_profiles
SET customer_tier = $2
WHERE id = $1
RETURNING *;

-- name: UpsertCustomerChannel :one
-- Resolves or creates the channel mapping, returning the customer profile.
-- Used by Synapse on every inbound message to identify the customer.
INSERT INTO customer_channels (
  customer_profile_id, channel_type, channel_external_id
) VALUES (
  $1, $2, $3
)
ON CONFLICT (customer_profile_id, channel_type, channel_external_id)
DO UPDATE SET customer_profile_id = EXCLUDED.customer_profile_id
RETURNING *;

-- name: GetCustomerByChannel :one
-- Primary Synapse lookup: find a customer from their external channel ID.
SELECT cp.*
FROM customer_profiles cp
JOIN customer_channels cc ON cc.customer_profile_id = cp.id
WHERE cc.channel_type = $1
  AND cc.channel_external_id = $2
LIMIT 1;

-- name: ListCustomers :many
SELECT 
    cp.id, 
    cp.unified_name, 
    cp.customer_tier, 
    cp.semantic_summary, 
    cp.created_at,
    (SELECT cc.channel_type 
     FROM customer_channels cc 
     WHERE cc.customer_profile_id = cp.id 
     ORDER BY cc.created_at ASC
     LIMIT 1)::varchar AS primary_channel
FROM customer_profiles cp
WHERE cp.business_id = $3
ORDER BY cp.created_at DESC
LIMIT $1 OFFSET $2;
