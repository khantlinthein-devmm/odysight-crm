-- 000011: real cleaner roster — link bookings to actual cleaners (FK),
-- with a role distinguishing the primary cleaner from additional crew.

CREATE TABLE IF NOT EXISTS booking_cleaners (
    id          BIGSERIAL PRIMARY KEY,
    booking_id  BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    cleaner_id  BIGINT NOT NULL REFERENCES cleaners(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'crew' CHECK (role IN ('primary', 'crew')),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (booking_id, cleaner_id)
);

CREATE INDEX IF NOT EXISTS idx_booking_cleaners_cleaner_id ON booking_cleaners (cleaner_id);
CREATE INDEX IF NOT EXISTS idx_booking_cleaners_booking_id ON booking_cleaners (booking_id);