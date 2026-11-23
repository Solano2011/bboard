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