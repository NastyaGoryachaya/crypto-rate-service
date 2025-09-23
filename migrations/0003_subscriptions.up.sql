CREATE TABLE IF NOT EXISTS subscriptions (
    chat_id          BIGINT PRIMARY KEY,
    interval_minutes INT     NOT NULL CHECK (interval_minutes > 0),
    enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    last_sent_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_enabled
    ON subscriptions (enabled);