CREATE TABLE IF NOT EXISTS cleaners (
    id           BIGSERIAL PRIMARY KEY,
    first_name   TEXT NOT NULL,
    last_name    TEXT NOT NULL,
    phone        TEXT NOT NULL,
    email        TEXT NOT NULL,
    skills       TEXT NOT NULL DEFAULT 'general',
    status       TEXT NOT NULL DEFAULT 'available'
        CHECK (status IN ('available', 'assigned', 'on_leave', 'inactive')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cleaners_status ON cleaners (status);
CREATE INDEX IF NOT EXISTS idx_cleaners_created_at ON cleaners (created_at DESC);
