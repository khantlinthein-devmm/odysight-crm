-- 000026: LINE identity on leads for OA chat → auto-lead.
-- LINE never shares phone numbers, so LINE-sourced leads start with phone ''
-- (staff collect it on first contact). The partial unique index dedupes
-- webhook retries and re-follows: one LINE user = one lead.

ALTER TABLE leads ADD COLUMN IF NOT EXISTS line_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE leads ADD COLUMN IF NOT EXISTS line_picture_url TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_leads_line_user
    ON leads (line_user_id) WHERE line_user_id <> '';
