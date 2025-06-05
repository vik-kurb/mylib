-- +goose Up
ALTER TABLE user_reading
ADD COLUMN start_date TIMESTAMP,
ADD COLUMN finish_date TIMESTAMP;

-- +goose Down
ALTER TABLE user_reading
DROP COLUMN start_date,
DROP COLUMN finish_date;
