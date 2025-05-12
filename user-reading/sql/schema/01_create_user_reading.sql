-- +goose Up
CREATE TYPE reading_status AS ENUM ('finished', 'reading', 'want_to_read');

CREATE TABLE IF NOT EXISTS user_reading(
    user_id UUID NOT NULL,
    book_id UUID NOT NULL,
    status reading_status NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, book_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_reading;
