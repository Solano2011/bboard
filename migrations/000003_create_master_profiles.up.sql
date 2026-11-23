CREATE TABLE IF NOT EXISTS master_profiles (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT       NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    headline         VARCHAR(200) NOT NULL,
    bio              TEXT         NOT NULL,
    category         VARCHAR(50)  NOT NULL,
    city             VARCHAR(100) NOT NULL DEFAULT '',
    experience_years INT          NOT NULL DEFAULT 0 CHECK (experience_years >= 0),
    min_price        BIGINT       NOT NULL DEFAULT 0 CHECK (min_price >= 0),
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_master_profiles_active_category
    ON master_profiles (is_active, category, created_at DESC);