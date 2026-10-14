-- 000048: checklist scoring, cleaner submit and office review.
-- item_type: 'check' (tick), 'score' (1-5) or 'pass_fail' (pass / fail / n/a).
-- A booking checklist copies item_type and require_photo from its template,
-- so later template edits never change a checklist already sent out.
ALTER TABLE checklist_template_items
    ADD COLUMN IF NOT EXISTS item_type TEXT NOT NULL DEFAULT 'check'
        CHECK (item_type IN ('check', 'score', 'pass_fail')),
    ADD COLUMN IF NOT EXISTS require_photo BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE booking_checklist_items
    ADD COLUMN IF NOT EXISTS item_type TEXT NOT NULL DEFAULT 'check'
        CHECK (item_type IN ('check', 'score', 'pass_fail')),
    ADD COLUMN IF NOT EXISTS require_photo BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS score SMALLINT CHECK (score BETWEEN 1 AND 5),
    ADD COLUMN IF NOT EXISTS result TEXT CHECK (result IN ('pass', 'fail', 'na')),
    ADD COLUMN IF NOT EXISTS completed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

-- Submitted checklists are locked for cleaners. The office approves them or
-- sends them back for rework (which clears submitted_at and unlocks).
ALTER TABLE booking_checklists
    ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS submitted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS review_status TEXT NOT NULL DEFAULT ''
        CHECK (review_status IN ('', 'approved', 'rework')),
    ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reviewed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_booking_checklists_submitted_at ON booking_checklists (submitted_at);
