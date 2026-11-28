BEGIN;
DROP TRIGGER IF EXISTS film_premiere_dates_sync ON film;
DO $$ BEGIN
    EXECUTE format('DROP FUNCTION IF EXISTS %I.sync_film_premiere_dates()',current_schema());
END $$;
DO $$ BEGIN
    IF to_regclass(format('%I.calendar_schema_migrations',current_schema())) IS NOT NULL THEN
        DELETE FROM calendar_schema_migrations WHERE version=20;
    END IF;
END $$;
COMMIT;
