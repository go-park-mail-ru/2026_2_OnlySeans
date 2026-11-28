BEGIN;
CREATE FUNCTION sync_film_premiere_dates() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    russia_id bigint;
BEGIN
    SELECT id INTO STRICT russia_id FROM country WHERE iso_code='RU';
    DELETE FROM film_release WHERE film_id=NEW.id
        AND (release_type='world_premiere' OR (release_type='cinema' AND country_id=russia_id));
    IF NEW.world_premiere_date IS NOT NULL THEN
        INSERT INTO film_release (film_id,country_id,release_type,release_date)
        VALUES (NEW.id,russia_id,'world_premiere',NEW.world_premiere_date);
    END IF;
    IF NEW.russia_premiere_date IS NOT NULL THEN
        INSERT INTO film_release (film_id,country_id,release_type,release_date)
        VALUES (NEW.id,russia_id,'cinema',NEW.russia_premiere_date);
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER film_premiere_dates_sync
    AFTER INSERT OR UPDATE OF world_premiere_date,russia_premiere_date ON film
    FOR EACH ROW EXECUTE FUNCTION sync_film_premiere_dates();
UPDATE film SET world_premiere_date=world_premiere_date;
COMMIT;
