-- name: GetUserByEmail :one
SELECT id, hashed_password FROM users
WHERE email = $1;