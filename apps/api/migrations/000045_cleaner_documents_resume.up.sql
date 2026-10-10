-- 000045: allow resumes (CVs) alongside identity / work-permit documents.

ALTER TABLE cleaner_documents DROP CONSTRAINT IF EXISTS cleaner_documents_doc_type_check;
ALTER TABLE cleaner_documents ADD CONSTRAINT cleaner_documents_doc_type_check
    CHECK (doc_type IN ('passport', 'visa', 'work_permit', 'pink_card', 'id_card', 'resume', 'other'));
