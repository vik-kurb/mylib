-- +goose Up
ALTER TABLE user_reading ADD COLUMN rating INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE user_reading DROP COLUMN rating;
