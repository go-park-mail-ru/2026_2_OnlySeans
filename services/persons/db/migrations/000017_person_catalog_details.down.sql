BEGIN;

ALTER TABLE file
    DROP COLUMN IF EXISTS source_credit,
    DROP COLUMN IF EXISTS source_url;

ALTER TABLE person
    DROP COLUMN IF EXISTS source_url,
    DROP COLUMN IF EXISTS biography,
    DROP COLUMN IF EXISTS birth_place,
    DROP COLUMN IF EXISTS death_date,
    DROP COLUMN IF EXISTS original_name;

COMMIT;
