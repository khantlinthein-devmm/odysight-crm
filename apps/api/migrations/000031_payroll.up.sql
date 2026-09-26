-- 000031: cleaner pay rates for payroll. Kept apart from cleaners so pay is
-- only visible to roles with payroll.read, never through the cleaner list.

CREATE TABLE IF NOT EXISTS cleaner_pay_rates (
    cleaner_id     BIGINT PRIMARY KEY REFERENCES cleaners(id) ON DELETE CASCADE,
    pay_type       TEXT NOT NULL CHECK (pay_type IN ('hourly', 'daily', 'per_job', 'monthly')),
    rate           NUMERIC(12,2) NOT NULL CHECK (rate >= 0),
    ot_multiplier  NUMERIC(4,2) NOT NULL DEFAULT 1.5 CHECK (ot_multiplier >= 1 AND ot_multiplier <= 5),
    standard_hours NUMERIC(4,2) NOT NULL DEFAULT 8 CHECK (standard_hours > 0 AND standard_hours <= 24),
    updated_by     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
