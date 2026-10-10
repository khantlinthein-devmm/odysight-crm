-- 000044: identity / work-permit documents per cleaner (passport, visa,
-- work permit, pink card). Document numbers are AES-GCM sealed by the API
-- (doc_number_enc); scanned files live encrypted on disk under UPLOAD_DIR.

CREATE TABLE IF NOT EXISTS cleaner_documents (
    id             BIGSERIAL PRIMARY KEY,
    cleaner_id     BIGINT NOT NULL REFERENCES cleaners(id) ON DELETE CASCADE,
    doc_type       TEXT NOT NULL CHECK (doc_type IN ('passport', 'visa', 'work_permit', 'pink_card', 'id_card', 'other')),
    doc_number_enc BYTEA,
    issue_date     DATE,
    expiry_date    DATE,
    notes          TEXT NOT NULL DEFAULT '',
    file_name      TEXT NOT NULL DEFAULT '',
    original_name  TEXT NOT NULL DEFAULT '',
    content_type   TEXT NOT NULL DEFAULT '',
    size_bytes     BIGINT NOT NULL DEFAULT 0,
    uploaded_by    BIGINT REFERENCES users(id) ON DELETE SET NULL,
    -- Highest expiry-reminder stage already sent (0 none, 1=60d, 2=30d, 3=7d, 4=expired).
    reminder_stage SMALLINT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cleaner_documents_cleaner ON cleaner_documents (cleaner_id);
CREATE INDEX IF NOT EXISTS idx_cleaner_documents_expiry ON cleaner_documents (expiry_date) WHERE expiry_date IS NOT NULL;

DROP TRIGGER IF EXISTS cleaner_documents_set_updated_at ON cleaner_documents;
CREATE TRIGGER cleaner_documents_set_updated_at BEFORE UPDATE ON cleaner_documents
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
