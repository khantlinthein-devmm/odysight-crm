CREATE TABLE IF NOT EXISTS applicants (
    id          BIGSERIAL PRIMARY KEY,
    first_name  TEXT NOT NULL,
    last_name   TEXT NOT NULL,
    email       TEXT NOT NULL,
    phone       TEXT NOT NULL,
    nationality TEXT NOT NULL,
    visa_type   TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'screening'
        CHECK (status IN ('screening', 'document_collection', 'submitted', 'processing', 'approved', 'rejected')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_applicants_status ON applicants (status);
CREATE INDEX IF NOT EXISTS idx_applicants_created_at ON applicants (created_at DESC);
