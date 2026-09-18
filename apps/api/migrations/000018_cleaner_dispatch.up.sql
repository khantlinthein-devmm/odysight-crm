-- 000018: cleaner dispatch — area-based job matching + live location.
-- Lets the cleaner mobile app go online with a GPS ping and an area
-- (e.g. "Bang Na"), and lets bookings carry an area so cleaners only see
-- jobs in their own area.

ALTER TABLE cleaners
    ADD COLUMN IF NOT EXISTS area TEXT NOT NULL DEFAULT '';
ALTER TABLE cleaners
    ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE cleaners
    ADD COLUMN IF NOT EXISTS lat DOUBLE PRECISION;
ALTER TABLE cleaners
    ADD COLUMN IF NOT EXISTS lng DOUBLE PRECISION;
ALTER TABLE cleaners
    ADD COLUMN IF NOT EXISTS is_online BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE cleaners
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS area TEXT NOT NULL DEFAULT '';

-- Backfill booking areas from the linked customer, so orders created via
-- the office before the mobile app existed still match by area.
UPDATE bookings b
SET area = c.area
FROM customers c
WHERE b.customer_id = c.id
  AND (b.area IS NULL OR b.area = '')
  AND c.area IS NOT NULL
  AND c.area <> '';

-- One login user maps to at most one cleaner profile (NULLs = unlinked rows).
CREATE UNIQUE INDEX IF NOT EXISTS idx_cleaners_user_id ON cleaners (user_id);
CREATE INDEX IF NOT EXISTS idx_cleaners_area ON cleaners (area);
CREATE INDEX IF NOT EXISTS idx_cleaners_online_area ON cleaners (is_online, area);
CREATE INDEX IF NOT EXISTS idx_bookings_area ON bookings (area);
CREATE INDEX IF NOT EXISTS idx_bookings_area_status ON bookings (area, status);
