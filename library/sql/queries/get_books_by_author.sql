-- name: GetBooksByAuthor :many
SELECT b.id, b.title FROM book_authors ba
JOIN books b ON ba.book_id = b.id
WHERE ba.author_id = $1;