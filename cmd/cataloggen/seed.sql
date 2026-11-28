-- Run against a freshly initialized fixture database after db/apply_all.sql.
-- Snapshot only public catalog reference tables, with stable ordering.
WITH tables(key, value) AS (VALUES
 ('genres', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM genre t)),
 ('countries', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM country t)),
 ('films', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM film t)),
 ('collections', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM collection t)),
 ('collection_films', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' ORDER BY collection_id, position) FROM collection_film t)),
 ('people', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM person t)),
 ('roles', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM role t)),
 ('permissions', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM permission t)),
 ('role_types', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY id) FROM role_type t)),
 ('role_permissions', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' ORDER BY role_id, permission_id) FROM role_permission t)),
 ('film_genres', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' ORDER BY film_id, genre_id) FROM film_genre t)),
 ('film_countries', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' ORDER BY film_id, country_id) FROM film_country t)),
 ('film_people', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' ORDER BY film_id,person_id,role_type_id) FROM film_person t)),
 ('similar_films', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' ORDER BY to_jsonb(t)::text) FROM similar_film t)),
 ('film_releases', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' ORDER BY to_jsonb(t)::text) FROM film_release t)),
 ('seasons', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY film_id,season_number) FROM season t)),
 ('episodes', (SELECT jsonb_agg(to_jsonb(t) - 'created_at' - 'updated_at' ORDER BY film_id,season_number,episode_number) FROM episode t))
)
SELECT jsonb_object_agg(key, coalesce(value,'[]'::jsonb)) FROM tables;
