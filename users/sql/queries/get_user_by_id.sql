-- name: GetUserById :one
SELECT login_name, email, birth_date FROM users
WHERE id = $1;
