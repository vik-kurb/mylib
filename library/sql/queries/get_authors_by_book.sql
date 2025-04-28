-- name: GetAuthorsByBook :many
SELECT author_id FROM book_authors
WHERE book_id = $1;