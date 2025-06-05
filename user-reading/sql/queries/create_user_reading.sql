-- name: CreateUserReading :exec
INSERT INTO user_reading (user_id, book_id, status, rating, start_date, finish_date)
VALUES (
    $1, $2, $3, $4, $5, $6
);