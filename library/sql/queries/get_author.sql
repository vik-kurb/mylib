-- name: GetAuthor :one
SELECT full_name, birth_date, death_date FROM authors
WHERE id = $1;