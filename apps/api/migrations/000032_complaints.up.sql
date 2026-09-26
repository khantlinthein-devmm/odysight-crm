-- 000032: customer complaints with an SLA and a one-click re-clean booking.

CREATE SEQUENCE IF NOT EXISTS complaint_seq;

CREATE TABLE IF NOT EXISTS complaints (
    id                 BIGSERIAL PRIMARY KEY,
    complaint_number   TEXT NOT NULL UNIQUE,
    customer_id        BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    site_id            BIGINT REFERENCES sites(id) ON DELETE SET NULL,
    booking_id         BIGINT REFERENCES bookings(id) ON DELETE SET NULL,
    category           TEXT NOT NULL CHECK (category IN
                         ('quality', 'missed_area', 'late', 'no_show', 'damage', 'staff_behaviour', 'other')),
    severity           TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    channel            TEXT NOT NULL DEFAULT 'phone' CHECK (channel IN ('phone', 'line', 'email', 'portal', 'in_person')),
    description        TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    due_at             TIMESTAMPTZ NOT NULL,
    resolution         TEXT NOT NULL DEFAULT '',
    resolved_at        TIMESTAMPTZ,
    resolved_by        BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reclean_booking_id BIGINT REFERENCES bookings(id) ON DELETE SET NULL,
    created_by         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_complaints_status_due ON complaints (status, due_at);
CREATE INDEX IF NOT EXISTS idx_complaints_customer ON complaints (customer_id);
CREATE INDEX IF NOT EXISTS idx_complaints_site ON complaints (site_id);
