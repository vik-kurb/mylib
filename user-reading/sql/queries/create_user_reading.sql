-- name: CreateUserReading :exec
INSERT INTO user_reading (user_id, book_id, status, rating)
VALUES (
    $1, $2, $3, $4
);