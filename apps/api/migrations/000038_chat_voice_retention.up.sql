-- 000038: chat voice retention.
-- Voice notes are deleted from disk 30 days after they were sent; the
-- message row stays (so the conversation still reads) and is marked expired.
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS file_expired BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX IF NOT EXISTS idx_chat_messages_files ON chat_messages (created_at) WHERE file_name <> '';
