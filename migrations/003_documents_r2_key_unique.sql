CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_collection_r2_key
    ON documents (collection_id, r2_key);
