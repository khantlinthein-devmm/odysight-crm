DROP INDEX IF EXISTS idx_booking_checklists_submitted_at;
ALTER TABLE booking_checklists
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by_user_id,
    DROP COLUMN IF EXISTS review_note,
    DROP COLUMN IF EXISTS review_status,
    DROP COLUMN IF EXISTS submitted_by_user_id,
    DROP COLUMN IF EXISTS submitted_at;
ALTER TABLE booking_checklist_items
    DROP COLUMN IF EXISTS completed_by_user_id,
    DROP COLUMN IF EXISTS result,
    DROP COLUMN IF EXISTS score,
    DROP COLUMN IF EXISTS require_photo,
    DROP COLUMN IF EXISTS item_type;
ALTER TABLE checklist_template_items
    DROP COLUMN IF EXISTS require_photo,
    DROP COLUMN IF EXISTS item_type;
