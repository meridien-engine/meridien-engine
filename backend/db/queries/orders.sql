-- name: CreateOrder :one
INSERT INTO orders (
  business_id, customer_id, total_price, status, source, notes
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1
LIMIT 1;

-- name: ListOrdersByCustomer :many
SELECT * FROM orders
WHERE customer_id = $1
ORDER BY created_at DESC;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2
WHERE id = $1
RETURNING *;

-- name: ListOrders :many
SELECT * FROM orders
WHERE business_id = $3
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateOrderItem :one
INSERT INTO order_items (
  order_id, product_id, sku, name, quantity, unit_price
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListOrderItems :many
SELECT * FROM order_items
WHERE order_id = $1;
