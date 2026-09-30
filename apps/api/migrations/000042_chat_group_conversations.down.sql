DROP TABLE IF EXISTS chat_reads;
DELETE FROM chat_conversations WHERE group_id IS NOT NULL;
ALTER TABLE chat_conversations DROP CONSTRAINT IF EXISTS chat_conversations_kind;
ALTER TABLE chat_conversations DROP COLUMN IF EXISTS group_id;
ALTER TABLE chat_conversations ALTER COLUMN user_a SET NOT NULL;
ALTER TABLE chat_conversations ALTER COLUMN user_b SET NOT NULL;
ALTER TABLE chat_conversations ADD CONSTRAINT chat_conversations_check CHECK (user_a < user_b);
