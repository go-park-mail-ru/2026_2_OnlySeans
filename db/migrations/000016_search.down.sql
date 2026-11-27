BEGIN;

DROP INDEX film_person_search_person_role_idx;
DROP INDEX person_search_original_name_idx;
DROP INDEX person_search_name_idx;
DROP INDEX film_search_original_title_idx;
DROP INDEX film_search_title_idx;
ALTER TABLE person DROP COLUMN original_name;
-- pg_trgm may be used elsewhere; do not remove the shared extension.

COMMIT;
