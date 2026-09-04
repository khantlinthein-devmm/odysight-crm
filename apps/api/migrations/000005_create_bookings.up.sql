CREATE TABLE IF NOT EXISTS bookings (
    id              BIGSERIAL PRIMARY KEY,
    booking_number  TEXT NOT NULL UNIQUE,
    customer_name   TEXT NOT NULL,
    service_type    TEXT NOT NULL
        CHECK (service_type IN ('house_cleaning', 'condo_cleaning', 'deep_cleaning',
                                'move_in_out', 'after_renovation', 'office_cleaning',
                                'junk_removal', 'aircon_service')),
    scheduled_for   TIMESTAMPTZ NOT NULL,
    duration_minutes INT NOT NULL DEFAULT 180,
    address         TEXT NOT NULL DEFAULT '',
    assigned_cleaner TEXT,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'confirmed', 'in_progress', 'completed', 'cancelled', 'no_show', 'rescheduled')),
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings (status);
CREATE INDEX IF NOT EXISTS idx_bookings_scheduled_for ON bookings (scheduled_for);
CREATE INDEX IF NOT EXISTS idx_bookings_created_at ON bookings (created_at DESC);
