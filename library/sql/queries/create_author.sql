-- name: CreateAuthor :one
INSERT INTO authors (id, full_name, birth_date, death_date, created_at, updated_at)
VALUES (
    gen_random_uuid(), $1, $2, $3, NOW(), NOW()
)
RETURNING id;