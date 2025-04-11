-- name: UpdateUser :one
UPDATE users SET login_name = $2, email = $3, birth_date = $4, hashed_password = $5, updated_at = NOW()
WHERE id = $1
RETURNING 1;