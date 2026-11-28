BEGIN;
DROP FUNCTION enqueue_detached_file() CASCADE;
DROP FUNCTION file_has_references(bigint);
DROP TABLE file_cleanup_queue;
DO $$ BEGIN
    IF to_regclass(format('%I.calendar_schema_migrations',current_schema())) IS NOT NULL THEN
        DELETE FROM calendar_schema_migrations WHERE version=22;
    END IF;
END $$;
COMMIT;
