-- +goose Up
-- Initial schema for Nouriko (portable between local PostgreSQL and Supabase).

CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY,
    profile_url TEXT NOT NULL DEFAULT '',
    height_cm NUMERIC(5, 2),
    weight_kg NUMERIC(5, 2),
    age SMALLINT,
    gender TEXT CHECK (gender IS NULL OR gender IN ('male', 'female')),
    activity_level TEXT CHECK (activity_level IS NULL OR activity_level IN ('light', 'moderate', 'heavy')),
    daily_calorie_target INTEGER,
    target_protein_g NUMERIC(6, 2),
    target_carbs_g NUMERIC(6, 2),
    target_fat_g NUMERIC(6, 2),
    target_fiber_g NUMERIC(6, 2),
    language TEXT NOT NULL DEFAULT 'id' CHECK (language IN ('id', 'en')),
    theme TEXT NOT NULL DEFAULT 'light' CHECK (theme IN ('light', 'dark')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS food_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price_idr INTEGER NOT NULL CHECK (price_idr >= 0),
    unit TEXT NOT NULL,
    category TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_food_prices_category ON food_prices (category);
CREATE INDEX IF NOT EXISTS idx_food_prices_price ON food_prices (price_idr);

CREATE TABLE IF NOT EXISTS saved_meals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    meal_date DATE NOT NULL,
    meal_time TEXT NOT NULL CHECK (meal_time IN ('breakfast', 'lunch', 'dinner', 'snack')),
    name TEXT NOT NULL,
    health_score SMALLINT NOT NULL CHECK (health_score BETWEEN 0 AND 100),
    calories INTEGER NOT NULL CHECK (calories >= 0),
    protein_g NUMERIC(6, 2) NOT NULL CHECK (protein_g >= 0),
    carbs_g NUMERIC(6, 2) NOT NULL CHECK (carbs_g >= 0),
    fat_g NUMERIC(6, 2) NOT NULL CHECK (fat_g >= 0),
    fiber_g NUMERIC(6, 2) NOT NULL CHECK (fiber_g >= 0),
    price_idr INTEGER CHECK (price_idr IS NULL OR price_idr >= 0),
    input_source TEXT NOT NULL CHECK (input_source IN ('ingredients', 'budget', 'photo')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_saved_meals_user_date ON saved_meals (user_id, meal_date DESC);

-- +goose StatementBegin
DO $$
BEGIN
    -- Enable Supabase Row Level Security automatically when running on Supabase.
    IF EXISTS (
        SELECT 1
        FROM pg_proc p
        JOIN pg_namespace n ON n.oid = p.pronamespace
        WHERE n.nspname = 'auth' AND p.proname = 'uid'
    ) THEN
        ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
        ALTER TABLE saved_meals ENABLE ROW LEVEL SECURITY;
        ALTER TABLE food_prices ENABLE ROW LEVEL SECURITY;

        DROP POLICY IF EXISTS profiles_owner_all ON profiles;
        CREATE POLICY profiles_owner_all ON profiles
            FOR ALL
            TO authenticated
            USING (auth.uid() = id)
            WITH CHECK (auth.uid() = id);

        DROP POLICY IF EXISTS saved_meals_owner_all ON saved_meals;
        CREATE POLICY saved_meals_owner_all ON saved_meals
            FOR ALL
            TO authenticated
            USING (auth.uid() = user_id)
            WITH CHECK (auth.uid() = user_id);

        DROP POLICY IF EXISTS food_prices_authenticated_read ON food_prices;
        CREATE POLICY food_prices_authenticated_read ON food_prices
            FOR SELECT
            TO authenticated
            USING (true);
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS saved_meals;
DROP TABLE IF EXISTS food_prices;
DROP TABLE IF EXISTS profiles;
