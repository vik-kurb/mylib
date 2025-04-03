-- +goose Up
CREATE TABLE IF NOT EXISTS authors(
    id UUID PRIMARY KEY,
    first_name TEXT NOT NULL,
    family_name TEXT NOT NULL,
    birth_date DATE,
    death_date DATE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS authors;
