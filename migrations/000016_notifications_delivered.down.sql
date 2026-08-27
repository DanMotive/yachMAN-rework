DROP INDEX IF EXISTS idx_notifications_undelivered;
ALTER TABLE notifications DROP COLUMN IF EXISTS delivered;
