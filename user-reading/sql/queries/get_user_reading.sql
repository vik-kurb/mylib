-- name: GetUserReading :many
SELECT book_id, status, rating, start_date, finish_date, created_at FROM user_reading
WHERE user_id = $1;