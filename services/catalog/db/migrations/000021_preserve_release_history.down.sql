BEGIN;
DO $$ BEGIN
    IF to_regclass(format('%I.calendar_schema_migrations', current_schema())) IS NOT NULL THEN
        DELETE FROM calendar_schema_migrations WHERE version=21;
    END IF;
END $$;
COMMIT;
