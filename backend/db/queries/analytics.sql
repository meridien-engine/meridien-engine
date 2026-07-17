-- name: GetRevenueLast7Days :many
WITH date_series AS (
    SELECT generate_series(
        current_date - interval '6 days',
        current_date,
        '1 day'::interval
    )::date AS day
)
SELECT 
    ds.day::date AS date,
    COALESCE(SUM(o.total_price), 0)::numeric AS total_revenue
FROM date_series ds
LEFT JOIN orders o ON ds.day = (o.created_at AT TIME ZONE 'UTC')::date
    AND o.business_id = $1
    AND o.status = 'completed'
GROUP BY ds.day
ORDER BY ds.day;

-- name: GetDashboardOverviewMetrics :one
SELECT 
    COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.total_price ELSE 0 END), 0)::numeric AS total_revenue,
    COUNT(o.id)::bigint AS orders_processed,
    COALESCE(
        COUNT(CASE WHEN o.source NOT IN ('portal', 'web') THEN 1 END)::numeric / 
        NULLIF(COUNT(o.id), 0)::numeric * 100, 
        0
    )::numeric AS interception_rate,
    (
        SELECT COUNT(t.id) 
        FROM interaction_traces t
        JOIN interaction_logs l ON t.interaction_log_id = l.id
        WHERE l.business_id = $1 AND t.hitl_status = 'pending'
    )::bigint AS pending_review
FROM orders o
WHERE o.business_id = $1;
