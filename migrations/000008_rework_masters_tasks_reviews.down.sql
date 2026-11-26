-- Откат миграции 000008
DROP TABLE IF EXISTS master_reviews;

-- Восстанавливаем task_bids
CREATE TABLE IF NOT EXISTS task_bids (
    id             BIGSERIAL PRIMARY KEY,
    task_id        BIGINT       NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    master_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    proposed_price BIGINT       NOT NULL CHECK (proposed_price >= 0),
    comment        TEXT         NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_task_bids_task_master UNIQUE (task_id, master_id)
);

CREATE INDEX IF NOT EXISTS idx_task_bids_task_id
    ON task_bids (task_id, created_at DESC);

INSERT INTO task_bids (task_id, master_id, proposed_price, comment, created_at)
SELECT
    tr.task_id,
    mp.user_id,
    COALESCE(tr.proposed_price, 0),
    tr.comment,
    tr.created_at
FROM task_responses tr
JOIN master_profiles mp ON mp.id = tr.master_profile_id
ON CONFLICT (task_id, master_id) DO NOTHING;

DROP TABLE IF EXISTS task_responses;

-- Восстанавливаем tasks.master_id
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS master_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;

UPDATE tasks t
SET master_id = mp.user_id
FROM master_profiles mp
WHERE t.assigned_master_id = mp.id;

DROP INDEX IF EXISTS idx_tasks_assigned_master_id;
ALTER TABLE tasks
    DROP COLUMN IF EXISTS assigned_master_id;

-- Восстанавливаем старую таблицу master_profiles
ALTER TABLE master_profiles RENAME TO master_profiles_new;

CREATE TABLE master_profiles (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT       NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    contact_phone    VARCHAR(30)  NOT NULL DEFAULT '',
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

INSERT INTO master_profiles (
    user_id, contact_phone, headline, bio, category, city, experience_years, min_price, is_active, created_at, updated_at
)
SELECT DISTINCT ON (mp.user_id)
    mp.user_id,
    mp.contact_phone,
    COALESCE(ms.title, mp.display_name),
    COALESCE(NULLIF(ms.experience_description, ''), mp.about),
    COALESCE(ms.category, 'other'),
    mp.city,
    0,
    COALESCE(ms.price, 0),
    COALESCE(ms.is_active, TRUE),
    mp.created_at,
    mp.updated_at
FROM master_profiles_new mp
LEFT JOIN master_services ms ON ms.master_profile_id = mp.id
ORDER BY mp.user_id, ms.created_at ASC;

DROP TABLE IF EXISTS master_services;
DROP TABLE IF EXISTS master_profiles_new;
