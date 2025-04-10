-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = NOW() 
where token = $1;
