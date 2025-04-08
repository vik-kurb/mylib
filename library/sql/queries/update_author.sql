-- name: UpdateAuthor :execrows
UPDATE authors SET
    first_name = $2,
    family_name = $3,
    birth_date = $4,
    death_date = $5,
    updated_at = NOW()
WHERE id = $1;