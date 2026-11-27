BEGIN;

ALTER TABLE person
    ADD COLUMN IF NOT EXISTS original_name text,
    ADD COLUMN IF NOT EXISTS death_date date,
    ADD COLUMN IF NOT EXISTS birth_place text,
    ADD COLUMN IF NOT EXISTS biography text,
    ADD COLUMN IF NOT EXISTS source_url text;

ALTER TABLE file
    ADD COLUMN IF NOT EXISTS source_url text,
    ADD COLUMN IF NOT EXISTS source_credit text;

COMMIT;
