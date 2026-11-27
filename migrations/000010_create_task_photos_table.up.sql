CREATE TABLE IF NOT EXISTS task_photos (
    id          BIGSERIAL    PRIMARY KEY,
    task_id     BIGINT       NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    storage_key VARCHAR(255) NOT NULL UNIQUE,
    mime_type   VARCHAR(50)  NOT NULL CHECK (mime_type IN ('image/jpeg', 'image/png', 'image/webp')),
    size_bytes  BIGINT       NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 5242880),
    sort_order  INT          NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_photos_task_id
    ON task_photos (task_id, sort_order ASC, id ASC);
