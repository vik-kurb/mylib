-- name: DeleteUserReading :exec
DELETE FROM user_reading
WHERE user_id = $1 AND book_id = $2;