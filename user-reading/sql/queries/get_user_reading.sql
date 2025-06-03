-- name: GetUserReading :many
SELECT book_id, status, rating FROM user_reading
WHERE user_id = $1;