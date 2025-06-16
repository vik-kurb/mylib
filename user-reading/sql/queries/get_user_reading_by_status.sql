-- name: GetUserReadingByStatus :many
SELECT book_id, rating, start_date, finish_date, created_at FROM user_reading
WHERE user_id = $1 AND status = $2;