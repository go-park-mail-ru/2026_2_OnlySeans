-- Каталог, который показывает фронтенд: 12 фильмов с постерами из static/posters,
-- их жанры и 4 подборки. Те же данные лежат в бэкенде в памяти (internal/films/seed.go),
-- миграция переносит их в базу, чтобы после переключения на PostgreSQL сайт не изменился.
-- «Побег из Шоушенка» и «Во все тяжкие» уже есть в 000008: им добавляются только постеры.
-- Постер хранится как storage_key 'posters/<имя>.svg', ссылка для фронтенда —
-- '/static/' || storage_key.

INSERT INTO genre (name, slug)
VALUES ('мультфильм', 'animation'),
       ('приключения', 'adventure'),
       ('семейный', 'family');

-- У сериалов длительность не задаётся: так требует film_duration_series_check.
INSERT INTO film (title, original_title, film_type, production_year, duration_min,
                  age_limit, description)
VALUES ('Зелёная миля', 'The Green Mile', 'movie', 1999, 189, 16,
        'Надзиратель блока смертников знакомится с заключённым, обладающим необычным даром.'),
       ('Форрест Гамп', 'Forrest Gump', 'movie', 1994, 142, 12,
        'История простодушного человека, который оказывается в центре главных событий эпохи.'),
       ('Интерстеллар', 'Interstellar', 'movie', 2014, 169, 16,
        'Группа исследователей отправляется через червоточину в поисках нового дома для человечества.'),
       ('Начало', 'Inception', 'movie', 2010, 148, 12,
        'Вор, крадущий идеи из снов, получает задание не украсть идею, а внедрить её.'),
       ('Остров проклятых', 'Shutter Island', 'movie', 2010, 138, 18,
        'Два пристава расследуют исчезновение пациентки из психиатрической клиники на острове.'),
       ('Король Лев', 'The Lion King', 'movie', 1994, 88, 0,
        'Львёнок Симба должен вернуть себе трон, который отнял его коварный дядя.'),
       ('Один дома', 'Home Alone', 'movie', 1990, 103, 0,
        'Восьмилетнего Кевина случайно забывают дома, и ему приходится защищать дом от грабителей.'),
       ('Иван Васильевич меняет профессию', NULL, 'movie', 1973, 88, 6,
        'Машина времени изобретателя Шурика меняет местами управдома и царя Ивана Грозного.'),
       ('Джентльмены', 'The Gentlemen', 'movie', 2019, 113, 18,
        'Наркобарон решает продать свой бизнес, и вокруг сделки начинается большая игра.'),
       ('Шерлок', 'Sherlock', 'series', 2010, NULL, 12,
        'Современная версия историй о Шерлоке Холмсе и докторе Ватсоне.');

INSERT INTO file (storage_key, mime_type, size_bytes)
VALUES ('posters/shawshank-redemption.svg', 'image/svg+xml', 1287),
       ('posters/green-mile.svg', 'image/svg+xml', 1093),
       ('posters/forrest-gump.svg', 'image/svg+xml', 1116),
       ('posters/interstellar.svg', 'image/svg+xml', 1319),
       ('posters/inception.svg', 'image/svg+xml', 1135),
       ('posters/shutter-island.svg', 'image/svg+xml', 1136),
       ('posters/lion-king.svg', 'image/svg+xml', 1018),
       ('posters/home-alone.svg', 'image/svg+xml', 1114),
       ('posters/ivan-vasilievich.svg', 'image/svg+xml', 1329),
       ('posters/the-gentlemen.svg', 'image/svg+xml', 1061),
       ('posters/sherlock.svg', 'image/svg+xml', 1137),
       ('posters/breaking-bad.svg', 'image/svg+xml', 1230);

-- Постеры из 000008 у «Шоушенка» и «Во все тяжкие» сдвигаются на 10 позиций,
-- чтобы первым шёл постер, файл которого есть в static/posters. Сдвиг на 10,
-- а не на 1: ограничение film_file_order_key проверяется сразу для каждой строки,
-- и сдвиг на 1 столкнул бы позиции 1 и 2.
UPDATE film_file
   SET sort_order = sort_order + 10
 WHERE file_role = 'poster'
   AND film_id IN (SELECT id FROM film
                    WHERE original_title IN ('The Shawshank Redemption', 'Breaking Bad'));

INSERT INTO film_file (film_id, file_id, file_role, sort_order)
SELECT f.id, fl.id, 'poster', 1
  FROM (VALUES ('Побег из Шоушенка', 'posters/shawshank-redemption.svg'),
               ('Зелёная миля', 'posters/green-mile.svg'),
               ('Форрест Гамп', 'posters/forrest-gump.svg'),
               ('Интерстеллар', 'posters/interstellar.svg'),
               ('Начало', 'posters/inception.svg'),
               ('Остров проклятых', 'posters/shutter-island.svg'),
               ('Король Лев', 'posters/lion-king.svg'),
               ('Один дома', 'posters/home-alone.svg'),
               ('Иван Васильевич меняет профессию', 'posters/ivan-vasilievich.svg'),
               ('Джентльмены', 'posters/the-gentlemen.svg'),
               ('Шерлок', 'posters/sherlock.svg'),
               ('Во все тяжкие', 'posters/breaking-bad.svg'))
           AS v (title, storage_key)
       JOIN film AS f ON f.title = v.title
       JOIN file AS fl ON fl.storage_key = v.storage_key;

INSERT INTO film_genre (film_id, genre_id)
SELECT f.id, g.id
  FROM (VALUES ('Зелёная миля', 'drama'), ('Зелёная миля', 'crime'),
               ('Форрест Гамп', 'drama'), ('Форрест Гамп', 'comedy'),
               ('Интерстеллар', 'sci-fi'), ('Интерстеллар', 'drama'),
               ('Интерстеллар', 'adventure'),
               ('Начало', 'sci-fi'), ('Начало', 'thriller'),
               ('Остров проклятых', 'thriller'), ('Остров проклятых', 'drama'),
               ('Король Лев', 'animation'), ('Король Лев', 'family'),
               ('Король Лев', 'drama'),
               ('Один дома', 'comedy'), ('Один дома', 'family'),
               ('Иван Васильевич меняет профессию', 'comedy'),
               ('Иван Васильевич меняет профессию', 'sci-fi'),
               ('Джентльмены', 'crime'), ('Джентльмены', 'comedy'),
               ('Шерлок', 'crime'), ('Шерлок', 'drama'), ('Шерлок', 'thriller'))
           AS v (title, slug)
       JOIN film AS f ON f.title = v.title
       JOIN genre AS g ON g.slug = v.slug;

INSERT INTO collection (title, slug, description)
VALUES ('Лучшие фильмы всех времён', 'best-of-all-time',
        'Фильмы, которые стоит посмотреть хотя бы раз.'),
       ('Смотрим всей семьёй', 'family', 'Добрые фильмы для вечера с близкими.'),
       ('Закрученный сюжет', 'mind-benders',
        'До последней минуты не угадаешь, чем всё кончится.'),
       ('Сериалы на выходные', 'weekend-series', 'Сериалы, от которых трудно оторваться.');

INSERT INTO collection_film (collection_id, film_id, position)
SELECT c.id, f.id, v.position
  FROM (VALUES ('best-of-all-time', 'Побег из Шоушенка', 1),
               ('best-of-all-time', 'Зелёная миля', 2),
               ('best-of-all-time', 'Форрест Гамп', 3),
               ('best-of-all-time', 'Интерстеллар', 4),
               ('best-of-all-time', 'Начало', 5),
               ('best-of-all-time', 'Король Лев', 6),
               ('family', 'Король Лев', 1),
               ('family', 'Один дома', 2),
               ('family', 'Форрест Гамп', 3),
               ('family', 'Иван Васильевич меняет профессию', 4),
               ('mind-benders', 'Начало', 1),
               ('mind-benders', 'Остров проклятых', 2),
               ('mind-benders', 'Интерстеллар', 3),
               ('mind-benders', 'Джентльмены', 4),
               ('weekend-series', 'Шерлок', 1),
               ('weekend-series', 'Во все тяжкие', 2))
           AS v (slug, title, position)
       JOIN collection AS c ON c.slug = v.slug
       JOIN film AS f ON f.title = v.title;
