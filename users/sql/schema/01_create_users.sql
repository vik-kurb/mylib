-- +goose Up
CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY,
    login_name TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    birth_date DATE,
    hashed_password TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    token TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;

DROP TABLE IF EXISTS users;
