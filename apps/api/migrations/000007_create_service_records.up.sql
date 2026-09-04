CREATE TABLE IF NOT EXISTS service_records (
    id             BIGSERIAL PRIMARY KEY,
    booking_number TEXT NOT NULL,
    cleaner_name   TEXT NOT NULL,
    service_type   TEXT NOT NULL,
    rating         SMALLINT CHECK (rating BETWEEN 1 AND 5),
    status         TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'completed', 'rescheduled', 'cancelled')),
    notes          TEXT NOT NULL DEFAULT '',
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_service_records_status ON service_records (status);
CREATE INDEX IF NOT EXISTS idx_service_records_created_at ON service_records (created_at DESC);
