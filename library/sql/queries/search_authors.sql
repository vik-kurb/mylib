-- name: SearchAuthors :many
SELECT id, full_name, ts_rank(tsv, plainto_tsquery('english', $1)) AS rank
FROM authors WHERE tsv @@ plainto_tsquery('english', $1)
ORDER BY rank DESC
LIMIT $2;
