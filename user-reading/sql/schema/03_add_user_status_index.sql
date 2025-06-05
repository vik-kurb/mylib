-- +goose Up
CREATE INDEX idx_user_reading_user_id_status ON user_reading(user_id, status);

-- +goose Down
DROP INDEX IF EXISTS idx_user_reading_user_id_status;
