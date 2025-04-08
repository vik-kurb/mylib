-- name: UpdateAuthor :execrows
UPDATE authors SET
    full_name = $2,
    birth_date = $3,
    death_date = $4,
    updated_at = NOW()
WHERE id = $1;