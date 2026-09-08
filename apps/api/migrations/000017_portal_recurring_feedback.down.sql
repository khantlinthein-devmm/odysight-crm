-- 000017 down: remove portal, recurring, feedback and notification-log schema.

DROP TABLE IF EXISTS notification_logs;
DROP TABLE IF EXISTS feedback;

DROP INDEX IF EXISTS idx_bookings_recurring_due;
DROP INDEX IF EXISTS idx_bookings_series_id;

ALTER TABLE bookings DROP COLUMN IF EXISTS series_id;
ALTER TABLE bookings DROP COLUMN IF EXISTS recurrence;
ALTER TABLE bookings DROP COLUMN IF EXISTS is_recurring;

ALTER TABLE customers DROP COLUMN IF EXISTS portal_enabled;
ALTER TABLE customers DROP COLUMN IF EXISTS password_hash;