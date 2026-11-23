DROP FUNCTION IF EXISTS anonymize_account(bigint);
ALTER TABLE account
    DROP CONSTRAINT IF EXISTS account_avatar_anonymized_check,
    DROP CONSTRAINT IF EXISTS account_avatar_file_id_fkey,
    DROP COLUMN IF EXISTS avatar_file_id;
DROP TABLE IF EXISTS file;
