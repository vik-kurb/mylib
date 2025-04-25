-- name: AddBookAuthors :exec
INSERT INTO book_authors (book_id, author_id)
SELECT UNNEST(@books::UUID[]), UNNEST(@authors::UUID[]);