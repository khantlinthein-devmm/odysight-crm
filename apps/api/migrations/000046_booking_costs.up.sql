-- 000046: per-booking job costs entered by the office (not payroll):
-- each cleaner's fee for this job, transportation and other costs.
-- Used for the booking profit estimate. Never shown to cleaners.

CREATE TABLE IF NOT EXISTS booking_costs (
    booking_id   BIGINT PRIMARY KEY REFERENCES bookings(id) ON DELETE CASCADE,
    -- {"<cleaner id>": fee, ...}
    cleaner_fees JSONB NOT NULL DEFAULT '{}'::jsonb,
    transport    NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (transport >= 0),
    other        NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (other >= 0),
    other_note   TEXT NOT NULL DEFAULT '',
    updated_by   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
