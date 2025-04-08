-- name: CreateAuthor :exec
INSERT INTO authors (id, first_name, family_name, birth_date, death_date, created_at, updated_at)
VALUES (
    gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW()
);