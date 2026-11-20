-- +goose Up
ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE profiles
    DROP COLUMN IF EXISTS display_name;
