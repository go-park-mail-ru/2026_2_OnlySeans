DELETE FROM account_ban;

DELETE FROM collection WHERE slug = 'films-about-freedom';
DELETE FROM film WHERE original_title IN ('The Shawshank Redemption', 'Breaking Bad', 'Dune: Part Three');
DELETE FROM person WHERE last_name IN ('Роббинс', 'Фриман', 'Дарабонт', 'Крэнстон');
UPDATE account SET avatar_file_id = NULL;
DELETE FROM film_rating_history;
DELETE FROM account WHERE username IN ('admin', 'moderator', 'editor', 'anna', 'ivan');
DELETE FROM file;
DELETE FROM role_type WHERE code IN ('actor', 'director', 'writer', 'producer', 'composer', 'operator');
DELETE FROM country WHERE iso_code IN ('US', 'RU', 'GB', 'FR', 'JP');
DELETE FROM genre WHERE slug IN ('drama', 'comedy', 'crime', 'sci-fi', 'thriller', 'action', 'romance', 'documentary');
DELETE FROM role_permission;
DELETE FROM permission;
DELETE FROM role;
