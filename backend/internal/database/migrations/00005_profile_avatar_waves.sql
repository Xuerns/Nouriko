-- +goose Up
UPDATE profiles
SET
    profile_url = REPLACE(profile_url, '/10.x/slice/svg', '/10.x/waves/svg'),
    updated_at = NOW()
WHERE profile_url LIKE 'https://api.dicebear.com/10.x/slice/svg%';

-- +goose Down
UPDATE profiles
SET
    profile_url = REPLACE(profile_url, '/10.x/waves/svg', '/10.x/slice/svg'),
    updated_at = NOW()
WHERE profile_url LIKE 'https://api.dicebear.com/10.x/waves/svg%';
