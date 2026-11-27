BEGIN;
DROP INDEX notification_release_once_idx;
ALTER TABLE notification DROP COLUMN notification_type, DROP COLUMN film_id;
DROP INDEX film_russia_premiere_idx;
ALTER TABLE film DROP COLUMN world_premiere_date, DROP COLUMN russia_premiere_date;
COMMIT;
