-- name: CreateUser :one
INSERT INTO users (id, login_name, email, birth_date, hashed_password, created_at, updated_at)
VALUES (
    gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW()
)
RETURNING id;