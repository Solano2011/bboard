CREATE TABLE IF NOT EXISTS tasks (
    id          BIGSERIAL PRIMARY KEY,
    author_id   BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    master_id   BIGINT       NULL REFERENCES users(id) ON DELETE SET NULL,
    title       VARCHAR(200) NOT NULL,
    description TEXT         NOT NULL,
    category    VARCHAR(50)  NOT NULL,
    city        VARCHAR(100) NOT NULL DEFAULT '',
    budget      BIGINT       NOT NULL DEFAULT 0 CHECK (budget >= 0),
    status      VARCHAR(20)  NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_category_created
    ON tasks (status, category, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_author_id
    ON tasks (author_id);