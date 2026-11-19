-- +goose Up
-- Local authentication tables for APP_ENV=local (portable to Supabase with deny-all RLS).

CREATE TABLE IF NOT EXISTS local_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    failed_login_attempts SMALLINT NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0),
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_local_users_email_lower ON local_users (LOWER(email));

CREATE TABLE IF NOT EXISTS local_auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES local_users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    purpose TEXT NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_local_auth_tokens_user_purpose ON local_auth_tokens (user_id, purpose);
CREATE INDEX IF NOT EXISTS idx_local_auth_tokens_expires_at ON local_auth_tokens (expires_at);

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
        ALTER TABLE local_users ENABLE ROW LEVEL SECURITY;
        ALTER TABLE local_auth_tokens ENABLE ROW LEVEL SECURITY;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS local_auth_tokens;
DROP TABLE IF EXISTS local_users;
