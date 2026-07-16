-- queries/secrets.sql

-- name: UpsertSecret :one
-- Inserts or updates a secret for a business.
-- The value must be encrypted BEFORE calling this query.
INSERT INTO system_secrets (business_id, key_name, encrypted_val)
VALUES ($1, $2, $3)
ON CONFLICT (business_id, key_name)
DO UPDATE SET encrypted_val = EXCLUDED.encrypted_val
RETURNING *;

-- name: GetSecret :one
-- Retrieves a single encrypted secret by business + key name.
SELECT * FROM system_secrets
WHERE business_id = $1
  AND key_name = $2;

-- name: ListSecretKeys :many
-- Lists all key names (NOT values) for a business — for the admin UI.
SELECT id, business_id, key_name, created_at, updated_at
FROM system_secrets
WHERE business_id = $1
ORDER BY key_name ASC;

-- name: DeleteSecret :exec
DELETE FROM system_secrets
WHERE business_id = $1
  AND key_name = $2;
