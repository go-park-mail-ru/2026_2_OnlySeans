BEGIN;
DROP TRIGGER IF EXISTS film_premiere_dates_sync ON film;
DO $$ BEGIN
    EXECUTE format('DROP FUNCTION IF EXISTS %I.sync_film_premiere_dates()',current_schema());
END $$;
DO $$ BEGIN
    IF to_regclass(format('%I.calendar_schema_migrations',current_schema())) IS NOT NULL THEN
        DELETE FROM calendar_schema_migrations WHERE version IN (19,20);
    END IF;
END $$;
DROP INDEX notification_release_once_idx;
ALTER TABLE notification DROP COLUMN notification_type, DROP COLUMN film_id;
DROP INDEX film_russia_premiere_idx;
ALTER TABLE film DROP COLUMN world_premiere_date, DROP COLUMN russia_premiere_date;
COMMIT;
