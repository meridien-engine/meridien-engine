-- name: ListKnowledgeNodes :many
SELECT
  id,
  source_name,
  substring(content from 1 for 200)::text as content_preview,
  created_at
FROM knowledge_nodes
WHERE business_id = $1
ORDER BY created_at DESC;

-- name: InsertKnowledgeNode :one
INSERT INTO knowledge_nodes (
  business_id,
  source_name,
  content,
  embedding
) VALUES (
  $1, $2, $3, $4::vector
)
RETURNING id;
