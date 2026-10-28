ALTER TABLE review
    DROP CONSTRAINT IF EXISTS review_version_check,
    DROP COLUMN IF EXISTS version;
