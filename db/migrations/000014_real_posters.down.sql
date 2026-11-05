UPDATE collection
   SET cover_file_id = (SELECT id FROM file WHERE storage_key = 'cover/films-about-freedom.jpg')
 WHERE slug = 'films-about-freedom';

DELETE FROM film_file
 WHERE file_id IN (SELECT id FROM file
                    WHERE external_url LIKE 'https://avatars.mds.yandex.net/get-kinopoisk-image/%');

UPDATE film_file SET sort_order = sort_order - 20 WHERE file_role = 'poster' AND sort_order > 20;

DELETE FROM file WHERE external_url LIKE 'https://avatars.mds.yandex.net/get-kinopoisk-image/%';
