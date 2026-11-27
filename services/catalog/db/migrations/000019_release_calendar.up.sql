BEGIN;
ALTER TABLE film ADD COLUMN world_premiere_date date
    CHECK (world_premiere_date BETWEEN DATE '1888-01-01' AND DATE '2200-01-01');
ALTER TABLE film ADD COLUMN russia_premiere_date date
    CHECK (russia_premiere_date BETWEEN DATE '1888-01-01' AND DATE '2200-01-01');
UPDATE film f SET
    world_premiere_date = (SELECT min(r.release_date) FROM film_release r
        WHERE r.film_id=f.id AND r.release_type='world_premiere'),
    russia_premiere_date = (SELECT min(r.release_date) FROM film_release r
        JOIN country c ON c.id=r.country_id
        WHERE r.film_id=f.id AND r.release_type='cinema' AND c.iso_code='RU');
CREATE INDEX film_russia_premiere_idx ON film (russia_premiere_date, id)
    WHERE russia_premiere_date IS NOT NULL;
ALTER TABLE notification ADD COLUMN film_id bigint REFERENCES film(id) ON DELETE SET NULL;
ALTER TABLE notification ADD COLUMN notification_type text;
CREATE UNIQUE INDEX notification_release_once_idx ON notification (account_id, film_id, notification_type)
    WHERE film_id IS NOT NULL;
COMMIT;
