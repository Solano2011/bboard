ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

ALTER TABLE users
    ADD CONSTRAINT chk_users_role CHECK (role IN ('user', 'moderator', 'admin'));

CREATE INDEX IF NOT EXISTS idx_users_role
    ON users (role);
