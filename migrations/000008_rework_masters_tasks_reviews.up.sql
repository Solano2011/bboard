-- 1. Переименовываем старую таблицу анкет для безопасного переноса данных
DROP INDEX IF EXISTS idx_master_profiles_active_category;
ALTER TABLE master_profiles RENAME TO master_profiles_old;

-- 2. Создаём новую таблицу профилей мастеров (1 профиль на аккаунт user_id)
CREATE TABLE master_profiles (
    id                     BIGSERIAL PRIMARY KEY,
    user_id                BIGINT        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name           VARCHAR(120)  NOT NULL,
    avatar_url             TEXT          NOT NULL DEFAULT '',
    about                  TEXT          NOT NULL DEFAULT '',
    city                   VARCHAR(100)  NOT NULL DEFAULT '',
    district               VARCHAR(100)  NOT NULL DEFAULT '',
    contact_phone          VARCHAR(30)   NOT NULL DEFAULT '',
    telegram               VARCHAR(100)  NOT NULL DEFAULT '',
    is_contacts_public     BOOLEAN       NOT NULL DEFAULT TRUE,
    rating_avg             NUMERIC(3,2)  NOT NULL DEFAULT 0,
    reviews_count          INT           NOT NULL DEFAULT 0 CHECK (reviews_count >= 0),
    completed_orders_count INT           NOT NULL DEFAULT 0 CHECK (completed_orders_count >= 0),
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Переносим профили из старой таблицы (группируя по user_id)
INSERT INTO master_profiles (
    user_id, display_name, about, city, contact_phone, is_contacts_public,
    completed_orders_count, created_at, updated_at
)
SELECT DISTINCT ON (mp.user_id)
    mp.user_id,
    COALESCE(NULLIF(TRIM(u.name), ''), 'Мастер #' || mp.user_id::text),
    mp.bio,
    mp.city,
    mp.contact_phone,
    TRUE,
    (
        SELECT COUNT(*)::INT
        FROM tasks t
        WHERE t.master_id = mp.user_id AND t.status = 'completed'
    ),
    mp.created_at,
    mp.updated_at
FROM master_profiles_old mp
JOIN users u ON u.id = mp.user_id
ORDER BY mp.user_id, mp.created_at ASC;

-- 3. Создаём таблицу анкет услуг мастера (до 3 активных на профиль, уникальная активная категория)
CREATE TABLE master_services (
    id                     BIGSERIAL PRIMARY KEY,
    master_profile_id      BIGINT       NOT NULL REFERENCES master_profiles(id) ON DELETE CASCADE,
    category               VARCHAR(50)  NOT NULL,
    title                  VARCHAR(200) NOT NULL,
    experience_description TEXT         NOT NULL DEFAULT '',
    price                  BIGINT       NOT NULL DEFAULT 0 CHECK (price >= 0),
    is_active              BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_master_services_active_category
    ON master_services (master_profile_id, category)
    WHERE is_active = TRUE;

CREATE INDEX idx_master_services_active_category
    ON master_services (is_active, category, created_at DESC);

-- Переносим услуги из старых анкет
INSERT INTO master_services (
    master_profile_id, category, title, experience_description, price, is_active, created_at, updated_at
)
SELECT
    new_mp.id,
    old_mp.category,
    old_mp.headline,
    old_mp.bio,
    old_mp.min_price,
    old_mp.is_active,
    old_mp.created_at,
    old_mp.updated_at
FROM master_profiles_old old_mp
JOIN master_profiles new_mp ON new_mp.user_id = old_mp.user_id
ON CONFLICT DO NOTHING;

-- 4. Обновляем таблицу tasks: добавляем assigned_master_id (ссылка на master_profiles.id)
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS assigned_master_id BIGINT NULL REFERENCES master_profiles(id) ON DELETE SET NULL;

UPDATE tasks t
SET assigned_master_id = mp.id
FROM master_profiles mp
WHERE t.master_id = mp.user_id;

ALTER TABLE tasks
    DROP COLUMN IF EXISTS master_id;

CREATE INDEX IF NOT EXISTS idx_tasks_assigned_master_id
    ON tasks (assigned_master_id);

-- 5. Создаём таблицу откликов task_responses и переносим данные из task_bids
CREATE TABLE task_responses (
    id                BIGSERIAL PRIMARY KEY,
    task_id           BIGINT      NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    master_profile_id BIGINT      NOT NULL REFERENCES master_profiles(id) ON DELETE CASCADE,
    proposed_price    BIGINT      NULL CHECK (proposed_price >= 0),
    comment           TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_task_responses_task_master UNIQUE (task_id, master_profile_id)
);

CREATE INDEX idx_task_responses_task_id
    ON task_responses (task_id, created_at DESC);

INSERT INTO task_responses (task_id, master_profile_id, proposed_price, comment, created_at)
SELECT
    tb.task_id,
    mp.id,
    tb.proposed_price,
    tb.comment,
    tb.created_at
FROM task_bids tb
JOIN master_profiles mp ON mp.user_id = tb.master_id
ON CONFLICT (task_id, master_profile_id) DO NOTHING;

DROP TABLE IF EXISTS task_bids;

-- 6. Создаём таблицу отзывов master_reviews (1 заказ = максимум 1 отзыв)
CREATE TABLE master_reviews (
    id                BIGSERIAL PRIMARY KEY,
    task_id           BIGINT      NOT NULL UNIQUE REFERENCES tasks(id) ON DELETE CASCADE,
    master_profile_id BIGINT      NOT NULL REFERENCES master_profiles(id) ON DELETE CASCADE,
    author_user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating            INT         NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment           TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_master_reviews_profile_id
    ON master_reviews (master_profile_id, created_at DESC);

-- 7. Удаляем временную таблицу старых анкет
DROP TABLE IF EXISTS master_profiles_old;
