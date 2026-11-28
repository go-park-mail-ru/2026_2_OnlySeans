BEGIN;
DROP TRIGGER IF EXISTS film_premiere_dates_sync ON film;
DO $$ BEGIN
    EXECUTE format('DROP FUNCTION IF EXISTS %I.sync_film_premiere_dates()', current_schema());
END $$;
COMMIT;
