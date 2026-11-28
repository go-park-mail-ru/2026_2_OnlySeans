\set ON_ERROR_STOP on
SELECT pg_advisory_lock(2026020019);
CREATE TABLE IF NOT EXISTS calendar_schema_migrations (
    version integer PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);
SELECT EXISTS (SELECT 1 FROM calendar_schema_migrations WHERE version=19) AS migrated \gset
\if :migrated
\else
SELECT EXISTS (SELECT 1 FROM information_schema.columns
    WHERE table_schema=current_schema() AND table_name='film' AND column_name='russia_premiere_date') AS existing \gset
\if :existing
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema()
        AND table_name='film' AND column_name='world_premiere_date')
    OR NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema()
        AND table_name='notification' AND column_name='notification_type')
    OR NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema()
        AND table_name='notification' AND column_name='film_id') THEN
        RAISE EXCEPTION 'incomplete calendar schema';
    END IF;
END $$;
\else
\ir ../services/catalog/db/migrations/000019_release_calendar.up.sql
\endif
INSERT INTO calendar_schema_migrations(version) VALUES (19);
\endif
SELECT EXISTS (SELECT 1 FROM calendar_schema_migrations WHERE version=20) AS migrated \gset
\if :migrated
\else
SELECT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='film'::regclass AND tgname='film_premiere_dates_sync') AS existing \gset
\if :existing
\else
\ir ../services/catalog/db/migrations/000020_release_date_sync.up.sql
\endif
INSERT INTO calendar_schema_migrations(version) VALUES (20);
\endif
SELECT pg_advisory_unlock(2026020019);
