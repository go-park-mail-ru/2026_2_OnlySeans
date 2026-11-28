BEGIN;
DROP TRIGGER IF EXISTS film_premiere_dates_sync ON film;
DO $$ BEGIN
    EXECUTE format('DROP FUNCTION IF EXISTS %I.sync_film_premiere_dates()',current_schema());
END $$;
DO $$ BEGIN
    IF to_regclass(format('%I.calendar_schema_migrations',current_schema())) IS NOT NULL THEN
        DELETE FROM calendar_schema_migrations WHERE version IN (19,20,21);
    END IF;
END $$;
CREATE TABLE IF NOT EXISTS film_calendar_date_archive (
    film_id bigint PRIMARY KEY REFERENCES film(id) ON DELETE CASCADE,
    world_premiere_date date,
    russia_premiere_date date
);
INSERT INTO film_calendar_date_archive SELECT id,world_premiere_date,russia_premiere_date FROM film
ON CONFLICT (film_id) DO UPDATE SET world_premiere_date=EXCLUDED.world_premiere_date,russia_premiere_date=EXCLUDED.russia_premiere_date;
DROP INDEX notification_release_once_idx;
ALTER TABLE notification DROP COLUMN notification_type, DROP COLUMN film_id;
DROP INDEX film_russia_premiere_idx;
ALTER TABLE film DROP COLUMN world_premiere_date, DROP COLUMN russia_premiere_date;
COMMIT;
