CREATE TABLE IF NOT EXISTS urls (
    hash         VARCHAR(16) PRIMARY KEY,
    original_url TEXT NOT NULL,
    user_id      UUID REFERENCES users(user_id) ON DELETE SET NULL,
    hit_count    BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_urls_user_id   ON urls(user_id);
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at);
