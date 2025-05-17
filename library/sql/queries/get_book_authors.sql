-- name: GetBookAuthors :many
SELECT a.full_name FROM book_authors ba
JOIN authors a ON ba.author_id = a.id
WHERE ba.book_id = $1;