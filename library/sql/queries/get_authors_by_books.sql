-- name: GetAuthorsByBooks :many
SELECT book_id, author_id FROM book_authors
WHERE book_id IN (SELECT UNNEST($1::UUID[]));