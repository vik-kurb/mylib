-- name: GetUserReadingByBook :one
SELECT status, rating, start_date, finish_date FROM user_reading
WHERE user_id = $1 AND book_id = $2;