-- name: CheckAuthors :many
SELECT id FROM authors
WHERE id = ANY(@ids::uuid[]);
