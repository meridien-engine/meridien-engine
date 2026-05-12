-- queries/memberships.sql

-- name: CreateMembership :one
INSERT INTO user_business_memberships (
  user_id,
  business_id,
  role,
  invited_by
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetMembership :one
SELECT * FROM user_business_memberships
WHERE user_id    = $1
  AND business_id = $2
  AND status     = 'active'
LIMIT 1;

-- name: ListUserBusinesses :many
SELECT
  b.*,
  m.role,
  m.status AS membership_status
FROM user_business_memberships m
JOIN businesses b ON b.id = m.business_id
WHERE m.user_id = $1
  AND m.status  = 'active'
  AND b.deleted_at IS NULL
ORDER BY b.name ASC;

-- name: ListBusinessMembers :many
SELECT
  u.id,
  u.email,
  u.first_name,
  u.last_name,
  m.role,
  m.status,
  m.created_at AS joined_at
FROM user_business_memberships m
JOIN users u ON u.id = m.user_id
WHERE m.business_id = $1
  AND u.deleted_at   IS NULL
ORDER BY m.created_at ASC;

-- name: UpdateMembershipRole :one
UPDATE user_business_memberships
SET role = $3
WHERE user_id    = $1
  AND business_id = $2
RETURNING *;

-- name: UpdateMembershipStatus :one
UPDATE user_business_memberships
SET status = $3
WHERE user_id    = $1
  AND business_id = $2
RETURNING *;

-- name: CreateJoinRequest :one
INSERT INTO join_requests (
  user_id,
  business_id,
  message,
  role
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetPendingJoinRequest :one
SELECT * FROM join_requests
WHERE user_id    = $1
  AND business_id = $2
  AND status     = 'pending'
LIMIT 1;

-- name: ListPendingJoinRequests :many
SELECT
  jr.*,
  u.email,
  u.first_name,
  u.last_name
FROM join_requests jr
JOIN users u ON u.id = jr.user_id
WHERE jr.business_id = $1
  AND jr.status      = 'pending'
ORDER BY jr.created_at ASC;

-- name: ReviewJoinRequest :one
UPDATE join_requests
SET
  status      = $2,
  reviewed_by = $3,
  reviewed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateInvitation :one
INSERT INTO invitations (
  business_id,
  email,
  role,
  token,
  invited_by,
  expires_at
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetInvitationByToken :one
SELECT * FROM invitations
WHERE token  = $1
  AND status = 'pending'
  AND expires_at > NOW()
LIMIT 1;

-- name: AcceptInvitation :one
UPDATE invitations
SET status = 'accepted'
WHERE token = $1
RETURNING *;

-- name: ExpireInvitation :exec
UPDATE invitations
SET status = 'expired'
WHERE id = $1;
