-- Add delivered status for notification delivery tracking
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS delivered BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_notifications_undelivered ON notifications (delivered, created_at)
    WHERE delivered = FALSE;
