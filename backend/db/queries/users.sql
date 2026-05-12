-- queries/users.sql
-- sqlc generates type-safe Go from these queries.
-- Run: sqlc generate (from backend/)
--
-- Convention:
--   :one   → returns a single row (error if not found)
--   :many  → returns a slice
--   :exec  → returns only error
--   :execrows → returns rows affected + error

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
  email,
  first_name,
  last_name,
  phone,
  password_hash
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: SoftDeleteUser :exec
UPDATE users
SET deleted_at = NOW()
WHERE id = $1;
