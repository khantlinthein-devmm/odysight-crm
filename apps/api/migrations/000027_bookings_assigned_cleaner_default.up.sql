-- 000027: bookings.assigned_cleaner must never be NULL.
-- Portal self-bookings inserted rows without that column, and scanBooking
-- reads it into a plain string, so a single NULL row returned 500 on every
-- bookings list/read endpoint. Backfill first, then enforce NOT NULL.

UPDATE bookings SET assigned_cleaner = '' WHERE assigned_cleaner IS NULL;

ALTER TABLE bookings ALTER COLUMN assigned_cleaner SET DEFAULT '';
ALTER TABLE bookings ALTER COLUMN assigned_cleaner SET NOT NULL;
