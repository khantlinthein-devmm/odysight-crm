-- 000017: Customer portal auth, recurring bookings, feedback & notification logs.

-- Customer self-service portal credentials.
ALTER TABLE customers ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS portal_enabled BOOLEAN NOT NULL DEFAULT false;

-- Recurring bookings: a booking that regenerates itself on a schedule.
-- series_id links every generated occurrence to its originating series.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS is_recurring BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS recurrence TEXT
    CHECK (recurrence IS NULL OR recurrence IN ('weekly', 'biweekly', 'monthly'));
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS series_id TEXT;

CREATE INDEX IF NOT EXISTS idx_bookings_series_id ON bookings (series_id);
CREATE INDEX IF NOT EXISTS idx_bookings_recurring_due
    ON bookings (is_recurring, scheduled_for) WHERE is_recurring;

-- Customer feedback / ratings on completed bookings.
CREATE TABLE IF NOT EXISTS feedback (
    id          BIGSERIAL PRIMARY KEY,
    booking_id  BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL,
    rating      INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment     TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_feedback_booking_id ON feedback (booking_id);
CREATE INDEX IF NOT EXISTS idx_feedback_rating ON feedback (rating);
CREATE INDEX IF NOT EXISTS idx_feedback_created_at ON feedback (created_at DESC);

-- Human-readable audit trail of every notification sent (email / SMS / WhatsApp).
CREATE TABLE IF NOT EXISTS notification_logs (
    id         BIGSERIAL PRIMARY KEY,
    channel    TEXT NOT NULL,
    event_type TEXT NOT NULL,
    recipient  TEXT NOT NULL,
    subject    TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL,
    error      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notification_logs_created_at ON notification_logs (created_at DESC);