-- Track processed Telegram updates to prevent duplicate execution
CREATE TABLE IF NOT EXISTS processed_updates (
    update_id   INTEGER PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-cleanup: drop entries older than 7 days
CREATE INDEX idx_processed_updates_age ON processed_updates (processed_at);
