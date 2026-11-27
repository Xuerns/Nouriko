-- +goose Up
-- OAuth identity and single-use replay protection for Nouriko (portable to Supabase with deny-all RLS).

ALTER TABLE local_users ALTER COLUMN password_hash DROP NOT NULL;

CREATE TABLE IF NOT EXISTS local_user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES local_users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('google')),
    provider_user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_user_id),
    UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_local_user_identities_user_id ON local_user_identities (user_id);

-- Shared single-use OAuth nonce replay store (used in both local and production modes).
CREATE TABLE IF NOT EXISTS oauth_consumed_nonces (
    nonce_hash TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_consumed_nonces_expires_at ON oauth_consumed_nonces (expires_at);

-- +goose StatementBegin
DO $$
BEGIN
    -- On Supabase, enable RLS with no anon/authenticated policies (deny-all for client keys).
    IF EXISTS (
        SELECT 1
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = 'auth' AND p.proname = 'uid'
    ) THEN
        ALTER TABLE local_user_identities ENABLE ROW LEVEL SECURITY;
        ALTER TABLE oauth_consumed_nonces ENABLE ROW LEVEL SECURITY;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS oauth_consumed_nonces;
DROP TABLE IF EXISTS local_user_identities;
DELETE FROM local_users WHERE password_hash IS NULL;
ALTER TABLE local_users ALTER COLUMN password_hash SET NOT NULL;
