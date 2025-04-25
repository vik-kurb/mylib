-- name: CreateBook :one
INSERT INTO books (id, title, created_at, updated_at)
VALUES (
    gen_random_uuid(), $1, NOW(), NOW()
)
RETURNING id;