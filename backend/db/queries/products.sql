-- name: CreateProduct :one
INSERT INTO products (
  business_id, sku, name, description, price, stock_qty
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetProductBySKU :one
SELECT * FROM products
WHERE business_id = $1
  AND sku = $2
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetProductByID :one
SELECT * FROM products
WHERE id = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products
WHERE deleted_at IS NULL
ORDER BY name ASC;

-- name: UpdateProduct :one
UPDATE products SET
  name        = COALESCE(sqlc.narg(name),        name),
  description = COALESCE(sqlc.narg(description), description),
  price       = COALESCE(sqlc.narg(price),       price),
  is_active   = COALESCE(sqlc.narg(is_active),   is_active)
WHERE id = sqlc.arg(id)
  AND deleted_at IS NULL
RETURNING *;

-- name: DecrementStock :one
-- Atomically decrements stock. Returns error (no rows) if insufficient stock.
UPDATE products
SET stock_qty = stock_qty - sqlc.arg(quantity)
WHERE id = sqlc.arg(id)
  AND stock_qty >= sqlc.arg(quantity)
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProduct :exec
UPDATE products
SET deleted_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;
