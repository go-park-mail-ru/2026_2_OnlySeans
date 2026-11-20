BEGIN;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE person ADD COLUMN original_name text;

CREATE INDEX film_search_title_idx
    ON film USING gin (lower(title) gin_trgm_ops);
CREATE INDEX film_search_original_title_idx
    ON film USING gin (lower(original_title) gin_trgm_ops);
CREATE INDEX person_search_name_idx
    ON person USING gin (lower(first_name || ' ' || last_name) gin_trgm_ops);
CREATE INDEX person_search_original_name_idx
    ON person USING gin (lower(original_name) gin_trgm_ops);
CREATE INDEX film_person_search_person_role_idx
    ON film_person (person_id, role_type_id);

COMMIT;
