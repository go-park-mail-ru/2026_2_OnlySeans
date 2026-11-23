DO $$
DECLARE
    incompatible_count bigint;
BEGIN
    LOCK TABLE account IN ACCESS EXCLUSIVE MODE;

    SELECT count(*) INTO incompatible_count
    FROM account
    WHERE username !~ '^[a-z0-9_]{3,32}$';

    IF incompatible_count > 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'Cannot roll back username rules without changing existing usernames',
            DETAIL = format('%s account(s) use names accepted only by the newer rules.', incompatible_count),
            HINT = 'Keep the current schema or prepare an explicit data migration before retrying. Usernames were not changed.';
    END IF;

    ALTER TABLE account
        DROP CONSTRAINT account_username_check,
        ADD CONSTRAINT account_username_check
            CHECK (username ~ '^[a-z0-9_]{3,32}$');

    COMMENT ON CONSTRAINT account_username_check ON account IS NULL;
END;
$$;
