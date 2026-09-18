-- 000020: multiple labeled phone numbers per cleaner.

CREATE TABLE IF NOT EXISTS cleaner_phones (
    id         BIGSERIAL PRIMARY KEY,
    cleaner_id BIGINT NOT NULL REFERENCES cleaners(id) ON DELETE CASCADE,
    label      TEXT NOT NULL DEFAULT 'Mobile',
    phone      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cleaner_phones_cleaner ON cleaner_phones (cleaner_id);
