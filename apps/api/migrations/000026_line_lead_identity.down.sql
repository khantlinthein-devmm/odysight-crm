DROP INDEX IF EXISTS idx_leads_line_user;
ALTER TABLE leads DROP COLUMN IF EXISTS line_picture_url;
ALTER TABLE leads DROP COLUMN IF EXISTS line_user_id;
