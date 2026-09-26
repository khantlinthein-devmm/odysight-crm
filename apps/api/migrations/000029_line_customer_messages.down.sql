DROP INDEX IF EXISTS idx_bookings_reminder_due;
ALTER TABLE bookings DROP COLUMN IF EXISTS completion_notified_at;
ALTER TABLE bookings DROP COLUMN IF EXISTS reminder_sent_at;
ALTER TABLE customers DROP COLUMN IF EXISTS line_user_id;
