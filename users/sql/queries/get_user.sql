-- name: GetUser :many
SELECT login_name, email FROM users
WHERE login_name = $1 OR email = $2;