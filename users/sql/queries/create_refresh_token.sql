-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (token, user_id, expires_at, created_at, updated_at)
VALUES (
    $1, $2, $3, NOW(), NOW()
);
