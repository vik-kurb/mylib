-- name: GetUserReadingByStatus :many
SELECT book_id, rating FROM user_reading
WHERE user_id = $1 AND status = $2;