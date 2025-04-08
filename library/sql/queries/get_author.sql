-- name: GetAuthor :one
SELECT first_name, family_name, birth_date, death_date FROM authors
WHERE id = $1;