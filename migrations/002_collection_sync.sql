ALTER TABLE collections
    ADD COLUMN IF NOT EXISTS embedding_model TEXT NOT NULL DEFAULT 'nomic-embed-text',
    ADD COLUMN IF NOT EXISTS tpuf_namespace TEXT,
    ADD COLUMN IF NOT EXISTS source_prefix TEXT;

UPDATE collections
SET tpuf_namespace = name
WHERE tpuf_namespace IS NULL OR tpuf_namespace = '';

UPDATE collections
SET source_prefix = 'collections/' || name
WHERE source_prefix IS NULL OR source_prefix = '';

ALTER TABLE collections
    ALTER COLUMN tpuf_namespace SET NOT NULL,
    ALTER COLUMN source_prefix SET NOT NULL;
