-- queries/businesses.sql

-- name: CreateBusiness :one
INSERT INTO businesses (
  name,
  slug,
  owner_id,
  category_id,
  contact_phone,
  contact_email
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetBusinessByID :one
SELECT * FROM businesses
WHERE id = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetBusinessBySlug :one
SELECT * FROM businesses
WHERE slug = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateBusiness :one
UPDATE businesses
SET
  name          = COALESCE($2, name),
  contact_phone = COALESCE($3, contact_phone),
  contact_email = COALESCE($4, contact_email),
  logo_url      = COALESCE($5, logo_url),
  status        = COALESCE($6, status)
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: ListBusinessCategories :many
SELECT * FROM business_categories
ORDER BY name ASC;
