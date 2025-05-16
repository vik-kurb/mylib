-- name: GetUserReading :many
SELECT book_id, status FROM user_reading
WHERE user_id = $1;