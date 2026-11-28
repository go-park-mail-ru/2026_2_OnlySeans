BEGIN;
CREATE TABLE file_cleanup_queue (
    file_id bigint PRIMARY KEY REFERENCES file(id) ON DELETE CASCADE,
    queued_at timestamptz NOT NULL DEFAULT now(),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    attempts integer NOT NULL DEFAULT 0
);

CREATE FUNCTION enqueue_detached_file() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE old_id bigint;
BEGIN
    old_id := (to_jsonb(OLD)->>TG_ARGV[0])::bigint;
    IF old_id IS NOT NULL AND (TG_OP='DELETE' OR
        old_id IS DISTINCT FROM (to_jsonb(NEW)->>TG_ARGV[0])::bigint) THEN
        INSERT INTO file_cleanup_queue(file_id) VALUES(old_id) ON CONFLICT DO NOTHING;
    END IF;
    RETURN NULL;
END;
$$;

CREATE FUNCTION file_has_references(target bigint) RETURNS boolean LANGUAGE plpgsql AS $$
DECLARE ref record; found boolean;
BEGIN
    FOR ref IN
        SELECT n.nspname,t.relname,a.attname
        FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid
        JOIN pg_namespace n ON n.oid=t.relnamespace
        JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum=c.conkey[1]
        WHERE c.contype='f' AND c.confrelid='file'::regclass
          AND c.conrelid<>'file_cleanup_queue'::regclass
    LOOP
        EXECUTE format('SELECT EXISTS(SELECT 1 FROM %I.%I WHERE %I=$1)',ref.nspname,ref.relname,ref.attname)
        INTO found USING target;
        IF found THEN RETURN true; END IF;
    END LOOP;
    RETURN false;
END;
$$;

DO $$ DECLARE ref record;
BEGIN
    FOR ref IN
        SELECT n.nspname,t.relname,a.attname
        FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid
        JOIN pg_namespace n ON n.oid=t.relnamespace
        JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum=c.conkey[1]
        WHERE c.contype='f' AND c.confrelid='file'::regclass
          AND c.conrelid<>'file_cleanup_queue'::regclass
    LOOP
        EXECUTE format('CREATE TRIGGER %I AFTER DELETE OR UPDATE OF %I ON %I.%I FOR EACH ROW EXECUTE FUNCTION enqueue_detached_file(%L)',
            'cleanup_'||ref.attname,ref.attname,ref.nspname,ref.relname,ref.attname);
    END LOOP;
END $$;
COMMIT;
