INSERT INTO role (code, title)
VALUES ('reader', 'читатель'),
       ('moderator', 'модератор'),
       ('editor', 'редактор каталога'),
       ('admin', 'администратор');

INSERT INTO permission (code, title)
VALUES ('review.write', 'писать рецензии'),
       ('review.moderate', 'модерировать рецензии'),
       ('message.moderate', 'модерировать сообщения чата'),
       ('catalog.edit', 'править каталог фильмов'),
       ('file.upload', 'загружать файлы'),
       ('account.manage', 'управлять учётными записями');

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
  FROM role AS r, permission AS p
 WHERE r.code = 'reader' AND p.code = 'review.write';

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
  FROM role AS r, permission AS p
 WHERE r.code = 'moderator'
   AND p.code IN ('review.write', 'review.moderate', 'message.moderate');

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id
  FROM role AS r, permission AS p
 WHERE r.code = 'editor'
   AND p.code IN ('review.write', 'catalog.edit', 'file.upload');

INSERT INTO role_permission (role_id, permission_id)
SELECT r.id, p.id FROM role AS r, permission AS p WHERE r.code = 'admin';

INSERT INTO genre (name, slug)
VALUES ('драма', 'drama'),
       ('комедия', 'comedy'),
       ('криминал', 'crime'),
       ('фантастика', 'sci-fi'),
       ('триллер', 'thriller'),
       ('боевик', 'action'),
       ('мелодрама', 'romance'),
       ('документальный', 'documentary');

INSERT INTO country (name, iso_code)
VALUES ('США', 'US'),
       ('Россия', 'RU'),
       ('Великобритания', 'GB'),
       ('Франция', 'FR'),
       ('Япония', 'JP');

INSERT INTO role_type (code, title)
VALUES ('actor', 'актёр'),
       ('director', 'режиссёр'),
       ('writer', 'сценарист'),
       ('producer', 'продюсер'),
       ('composer', 'композитор'),
       ('operator', 'оператор');

INSERT INTO account (role_id, email, username, password_hash, first_name)
SELECT r.id, v.email, v.username,
       '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', v.first_name
  FROM (VALUES ('admin@kinopoisk.local', 'admin', 'Администратор', 'admin'),
               ('moderator@kinopoisk.local', 'moderator', 'Модератор', 'moderator'),
               ('editor@kinopoisk.local', 'editor', 'Редактор', 'editor'),
               ('anna@example.com', 'anna', 'Анна', 'reader'),
               ('ivan@example.com', 'ivan', 'Иван', 'reader'))
           AS v (email, username, first_name, role_code)
       JOIN role AS r ON r.code = v.role_code;

INSERT INTO file (storage_key, mime_type, size_bytes, created_by)
SELECT v.storage_key, v.mime_type, v.size_bytes, a.id
  FROM (VALUES ('poster/shawshank-1.jpg', 'image/jpeg', 184320),
               ('poster/shawshank-2.jpg', 'image/jpeg', 176128),
               ('still/shawshank-roof.jpg', 'image/jpeg', 240640),
               ('poster/breaking-bad-1.jpg', 'image/jpeg', 198656),
               ('person/freeman.jpg', 'image/jpeg', 95232),
               ('cover/films-about-freedom.jpg', 'image/jpeg', 132096),
               ('avatar/anna.png', 'image/png', 40960))
           AS v (storage_key, mime_type, size_bytes)
       JOIN account AS a ON a.username = 'editor';

INSERT INTO file (external_url, mime_type, created_by)
SELECT v.external_url, 'video/mp4', a.id
  FROM (VALUES ('https://video.example.com/shawshank-trailer-1'),
               ('https://video.example.com/shawshank-trailer-2'))
           AS v (external_url)
       JOIN account AS a ON a.username = 'editor';

UPDATE account AS a
   SET avatar_file_id = f.id
  FROM file AS f
 WHERE a.username = 'anna' AND f.storage_key = 'avatar/anna.png';

INSERT INTO film (title, original_title, film_type, production_year, duration_min,
                  age_limit, description)
VALUES ('Побег из Шоушенка', 'The Shawshank Redemption', 'movie', 1994, 142, 16,
        'Несправедливо осуждённый банкир проводит в тюрьме Шоушенк два десятилетия, '
        'сохраняя надежду и обретая друзей.'),
       ('Дюна: Часть третья', 'Dune: Part Three', 'movie', 2026, 165, 12,
        'Третья часть экранизации романа Фрэнка Герберта.');

INSERT INTO film (title, original_title, film_type, production_year, age_limit,
                  description)
VALUES ('Во все тяжкие', 'Breaking Bad', 'series', 2008, 18,
        'Школьный учитель химии узнаёт о смертельном диагнозе и начинает '
        'производить метамфетамин, чтобы обеспечить семью.');

INSERT INTO film_file (film_id, file_id, file_role, sort_order)
SELECT f.id, v.file_id, v.file_role, v.sort_order
  FROM film AS f
       JOIN (SELECT id AS file_id, 'poster' AS file_role, 1 AS sort_order
               FROM file WHERE storage_key = 'poster/shawshank-1.jpg'
              UNION ALL
             SELECT id, 'poster', 2 FROM file WHERE storage_key = 'poster/shawshank-2.jpg'
              UNION ALL
             SELECT id, 'still', 1 FROM file WHERE storage_key = 'still/shawshank-roof.jpg'
              UNION ALL
             SELECT id, 'trailer', 1
               FROM file WHERE external_url = 'https://video.example.com/shawshank-trailer-1'
              UNION ALL
             SELECT id, 'trailer', 2
               FROM file WHERE external_url = 'https://video.example.com/shawshank-trailer-2'
            ) AS v ON true
 WHERE f.original_title = 'The Shawshank Redemption';

INSERT INTO film_file (film_id, file_id, file_role, sort_order)
SELECT f.id, fl.id, 'poster', 1
  FROM film AS f, file AS fl
 WHERE f.original_title = 'Breaking Bad'
   AND fl.storage_key = 'poster/breaking-bad-1.jpg';

INSERT INTO film_genre (film_id, genre_id)
SELECT f.id, g.id
  FROM film AS f
       JOIN genre AS g ON g.slug IN ('drama', 'crime')
 WHERE f.original_title = 'The Shawshank Redemption';

INSERT INTO film_genre (film_id, genre_id)
SELECT f.id, g.id
  FROM film AS f
       JOIN genre AS g ON g.slug IN ('drama', 'crime', 'thriller')
 WHERE f.original_title = 'Breaking Bad';

INSERT INTO film_genre (film_id, genre_id)
SELECT f.id, g.id
  FROM film AS f
       JOIN genre AS g ON g.slug = 'sci-fi'
 WHERE f.original_title = 'Dune: Part Three';

INSERT INTO film_country (film_id, country_id)
SELECT f.id, c.id
  FROM film AS f
       JOIN country AS c ON c.iso_code = 'US'
 WHERE f.original_title IN ('The Shawshank Redemption', 'Breaking Bad',
                            'Dune: Part Three');

INSERT INTO similar_film (film_id, similar_film_id)
SELECT least(a.id, b.id), greatest(a.id, b.id)
  FROM film AS a, film AS b
 WHERE a.original_title = 'The Shawshank Redemption'
   AND b.original_title = 'Breaking Bad';

INSERT INTO film_release (film_id, country_id, release_type, release_date)
SELECT f.id, c.id, 'world_premiere', DATE '1994-09-10'
  FROM film AS f, country AS c
 WHERE f.original_title = 'The Shawshank Redemption' AND c.iso_code = 'US';

INSERT INTO film_release (film_id, country_id, release_type, release_date)
SELECT f.id, c.id, 'cinema', v.release_date
  FROM film AS f, country AS c,
       (VALUES (DATE '1995-02-16'), (DATE '2024-09-19')) AS v (release_date)
 WHERE f.original_title = 'The Shawshank Redemption' AND c.iso_code = 'RU';

INSERT INTO film_release (film_id, country_id, release_type, release_date)
SELECT f.id, c.id, 'cinema', DATE '2026-12-17'
  FROM film AS f, country AS c
 WHERE f.original_title = 'Dune: Part Three' AND c.iso_code = 'RU';

INSERT INTO season (film_id, season_number, title)
SELECT id, 1, 'Сезон 1' FROM film WHERE original_title = 'Breaking Bad';

INSERT INTO episode (film_id, season_number, episode_number, title, duration_min,
                     release_date)
SELECT f.id, 1, v.episode_number, v.title, v.duration_min, v.release_date
  FROM film AS f,
       (VALUES (1, 'Пилот', 58::smallint, DATE '2008-01-20'),
               (2, 'Кошкин угол', 48::smallint, DATE '2008-01-27'))
           AS v (episode_number, title, duration_min, release_date)
 WHERE f.original_title = 'Breaking Bad';

INSERT INTO person (first_name, last_name, birth_date)
VALUES ('Тим', 'Роббинс', DATE '1958-10-16'),
       ('Морган', 'Фриман', DATE '1937-06-01'),
       ('Фрэнк', 'Дарабонт', DATE '1959-01-28'),
       ('Брайан', 'Крэнстон', DATE '1956-03-07');

UPDATE person AS p
   SET photo_file_id = f.id
  FROM file AS f
 WHERE p.last_name = 'Фриман' AND f.storage_key = 'person/freeman.jpg';

INSERT INTO film_person (film_id, person_id, role_type_id, character_name)
SELECT f.id, p.id, r.id, 'Энди Дюфрейн'
  FROM film AS f, person AS p, role_type AS r
 WHERE f.original_title = 'The Shawshank Redemption'
   AND p.last_name = 'Роббинс' AND r.code = 'actor';

INSERT INTO film_person (film_id, person_id, role_type_id, character_name)
SELECT f.id, p.id, r.id, 'Эллис Бойд Реддинг'
  FROM film AS f, person AS p, role_type AS r
 WHERE f.original_title = 'The Shawshank Redemption'
   AND p.last_name = 'Фриман' AND r.code = 'actor';

INSERT INTO film_person (film_id, person_id, role_type_id, character_name)
SELECT f.id, p.id, r.id, NULL
  FROM film AS f, person AS p, role_type AS r
 WHERE f.original_title = 'The Shawshank Redemption'
   AND p.last_name = 'Дарабонт' AND r.code = 'director';

INSERT INTO film_person (film_id, person_id, role_type_id, character_name)
SELECT f.id, p.id, r.id, 'Уолтер Уайт'
  FROM film AS f, person AS p, role_type AS r
 WHERE f.original_title = 'Breaking Bad'
   AND p.last_name = 'Крэнстон' AND r.code = 'actor';

INSERT INTO film_rating (account_id, film_id, score)
SELECT a.id, f.id, 10
  FROM account AS a, film AS f
 WHERE a.username = 'anna' AND f.original_title = 'The Shawshank Redemption';

INSERT INTO film_rating (account_id, film_id, score)
SELECT a.id, f.id, 9
  FROM account AS a, film AS f
 WHERE a.username = 'ivan' AND f.original_title = 'The Shawshank Redemption';

UPDATE film_rating AS r
   SET score = 10
  FROM account AS a, film AS f
 WHERE r.account_id = a.id AND r.film_id = f.id
   AND a.username = 'ivan' AND f.original_title = 'The Shawshank Redemption';

INSERT INTO review (account_id, film_id, title, content, contains_spoilers)
SELECT a.id, f.id, 'Надежда — хорошая вещь',
       'Фильм о том, что человека нельзя лишить внутренней свободы. '
       'Актёрская работа Тима Роббинса и Моргана Фримана держит внимание '
       'все два с лишним часа, а финал остаётся одним из самых сильных в кино.',
       false
  FROM account AS a, film AS f
 WHERE a.username = 'anna' AND f.original_title = 'The Shawshank Redemption';

INSERT INTO moderation_task (review_id, status, source, model_name, model_score,
                             reason, decided_at)
SELECT r.id, 'approved', 'auto', 'toxicity-ru-v1', 0.031,
       'Токсичности не обнаружено', now()
  FROM review AS r;

INSERT INTO folder (account_id, title, folder_type)
SELECT a.id, t.title, t.folder_type
  FROM account AS a
       CROSS JOIN (VALUES ('Избранное', 'favorite'),
                          ('Буду смотреть', 'watch_later'),
                          ('Просмотренные', 'watched'),
                          ('Любимые персоны', 'person')) AS t (title, folder_type);

INSERT INTO folder_item (folder_id, film_id)
SELECT fo.id, fi.id
  FROM folder AS fo
       JOIN account AS a ON a.id = fo.account_id
       JOIN film AS fi ON fi.original_title = 'The Shawshank Redemption'
 WHERE a.username = 'anna' AND fo.folder_type = 'favorite';

INSERT INTO folder_item (folder_id, person_id)
SELECT fo.id, p.id
  FROM folder AS fo
       JOIN account AS a ON a.id = fo.account_id
       JOIN person AS p ON p.last_name = 'Фриман'
 WHERE a.username = 'anna' AND fo.folder_type = 'person';

INSERT INTO release_subscription (account_id, film_id)
SELECT a.id, f.id
  FROM account AS a, film AS f
 WHERE a.username = 'anna' AND f.original_title = 'Dune: Part Three';

INSERT INTO collection (title, slug, description, cover_file_id)
SELECT 'Фильмы о свободе', 'films-about-freedom',
       'Подборка о тех, кто не сдаётся.', f.id
  FROM file AS f
 WHERE f.storage_key = 'cover/films-about-freedom.jpg';

INSERT INTO collection_film (collection_id, film_id, position)
SELECT c.id, f.id, 1
  FROM collection AS c, film AS f
 WHERE c.slug = 'films-about-freedom'
   AND f.original_title = 'The Shawshank Redemption';

INSERT INTO collection_film (collection_id, film_id, position)
SELECT c.id, f.id, 2
  FROM collection AS c, film AS f
 WHERE c.slug = 'films-about-freedom' AND f.original_title = 'Breaking Bad';

INSERT INTO notification (account_id, message)
SELECT a.id, 'Фильм "Дюна: Часть третья" выходит в прокат 17 декабря.'
  FROM account AS a
 WHERE a.username = 'anna';

INSERT INTO film_discussion (film_id)
SELECT id FROM film WHERE original_title = 'The Shawshank Redemption';

INSERT INTO discussion_message (discussion_id, account_id, message_text)
SELECT d.id, a.id, 'Пересматриваю каждый год, и каждый раз как в первый.'
  FROM film_discussion AS d, account AS a
 WHERE a.username = 'anna';

INSERT INTO moderation_task (message_id, status, source, model_name, model_score)
SELECT m.id, 'pending', 'auto', 'toxicity-ru-v1', 0.412
  FROM discussion_message AS m;

INSERT INTO account_ban (account_id, moderator_id, reason, active_during)
SELECT target.id, moder.id, 'Оскорбления в обсуждении фильма',
       tstzrange(now(), now() + interval '7 days')
  FROM account AS target, account AS moder
 WHERE target.username = 'ivan' AND moder.username = 'moderator';
