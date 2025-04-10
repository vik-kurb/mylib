-- name: GetUserByRefreshToken :one
SELECT user_id FROM refresh_tokens
WHERE token = $1 AND expires_at > NOW() AND revoked_at is NULL;