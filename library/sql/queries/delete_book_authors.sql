-- name: DeleteBookAuthors :exec
DELETE FROM book_authors
WHERE book_id = $1 AND author_id IN (SELECT UNNEST(@authors::UUID[]));
