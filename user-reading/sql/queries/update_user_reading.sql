-- name: UpdateUserReading :one
UPDATE user_reading SET status = $3
WHERE user_id = $1 AND book_id = $2
RETURNING 1;