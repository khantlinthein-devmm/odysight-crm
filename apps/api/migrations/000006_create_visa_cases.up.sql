CREATE TABLE IF NOT EXISTS visa_cases (
    id             BIGSERIAL PRIMARY KEY,
    case_number    TEXT NOT NULL UNIQUE,
    applicant_name TEXT NOT NULL,
    visa_type      TEXT NOT NULL,
    destination    TEXT NOT NULL,
    assigned_to    TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'in_review', 'submitted', 'additional_docs_required', 'approved', 'rejected', 'closed')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_visa_cases_status ON visa_cases (status);
CREATE INDEX IF NOT EXISTS idx_visa_cases_created_at ON visa_cases (created_at DESC);
