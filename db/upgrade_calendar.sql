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
SELECT EXISTS (SELECT 1 FROM calendar_schema_migrations WHERE version=21) AS migrated \gset
\if :migrated
\else
\ir ../services/catalog/db/migrations/000021_preserve_release_history.up.sql
INSERT INTO calendar_schema_migrations(version) VALUES (21);
\endif
SELECT EXISTS (SELECT 1 FROM calendar_schema_migrations WHERE version=22) AS migrated \gset
\if :migrated
\else
SELECT to_regclass(format('%I.file_cleanup_queue',current_schema())) IS NOT NULL AS existing \gset
\if :existing
\else
\ir migrations/000022_file_cleanup.up.sql
\endif
INSERT INTO calendar_schema_migrations(version) VALUES (22);
\endif
SELECT pg_advisory_unlock(2026020019);
