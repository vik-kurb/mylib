-- name: GetAuthorsNamesByBooks :many
SELECT ba.book_id, a.full_name FROM book_authors ba
JOIN authors a ON ba.author_id = a.id
WHERE book_id IN (SELECT UNNEST($1::UUID[]));