-- name: GetLiveFeed :many
SELECT 
    id,
    'interaction'::text AS item_type,
    channel::text AS title_meta,
    inbound_msg::text AS description,
    created_at
FROM interaction_logs
WHERE interaction_logs.business_id = @business_id::uuid
UNION ALL
SELECT 
    id,
    'order'::text AS item_type,
    status::text AS title_meta,
    CAST(total_price AS TEXT) AS description,
    created_at
FROM orders
WHERE orders.business_id = @business_id::uuid
ORDER BY created_at DESC
LIMIT 15;
