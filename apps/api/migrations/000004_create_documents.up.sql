CREATE TABLE IF NOT EXISTS documents (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    type           TEXT NOT NULL,
    applicant_name TEXT NOT NULL,
    file_size_kb   BIGINT NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'verified', 'rejected', 'expired')),
    uploaded_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_documents_status ON documents (status);
CREATE INDEX IF NOT EXISTS idx_documents_uploaded_at ON documents (uploaded_at DESC);
