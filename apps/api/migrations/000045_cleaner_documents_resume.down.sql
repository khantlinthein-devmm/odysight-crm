DELETE FROM cleaner_documents WHERE doc_type = 'resume';
ALTER TABLE cleaner_documents DROP CONSTRAINT IF EXISTS cleaner_documents_doc_type_check;
ALTER TABLE cleaner_documents ADD CONSTRAINT cleaner_documents_doc_type_check
    CHECK (doc_type IN ('passport', 'visa', 'work_permit', 'pink_card', 'id_card', 'other'));
