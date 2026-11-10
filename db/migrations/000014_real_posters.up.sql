-- Настоящие постеры фильмов вместо заглушек: ссылки на изображения Кинопоиска
-- (avatars.mds.yandex.net, размер 600x900). Файлы лежат на стороне Кинопоиска,
-- в базе хранится только внешняя ссылка — поле file.external_url.
-- Номера фильмов Кинопоиска взяты из Викиданных (свойство P2603).
-- Заглушки из static/posters и постеры из 000008 остаются запасными:
-- они сдвигаются на 20 позиций, а первым идёт настоящий постер.

INSERT INTO file (external_url, mime_type)
VALUES ('https://avatars.mds.yandex.net/get-kinopoisk-image/1599028/0b76b2a2-d1c7-4f04-a284-80ff7bb709a4/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/4483445/556cc7b7-d4cc-4b12-ada6-126238fdfd8a/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1900788/fb35416f-3b0d-4b96-bc65-cf6923f9e329/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1599028/4057c4b8-8208-4a04-b169-26b0661453e3/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1599028/3560b757-9b95-45ec-af8c-623972370f9d/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1600647/430042eb-ee69-4818-aed0-a312400a26bf/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1629390/8ab9a119-dd74-44f0-baec-0629797483d7/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/4303601/617303b7-cfa7-4273-bd1d-63974bf68927/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1704946/60aa1abc-b754-4817-ad9c-0bcda427a12b/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/6201401/022a58e3-5b9b-411b-bfb3-09fedb700401/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/6201401/a7ef44b8-1983-4992-a889-da6f87a3f559/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1599028/637271d5-61b4-4e46-ac83-6d07494c7645/600x900', 'image/jpeg'),
       ('https://avatars.mds.yandex.net/get-kinopoisk-image/1629390/f28c1ea2-47b0-49d5-b11c-9608744f0233/600x900', 'image/jpeg');

-- Сдвиг на 20, а не на 1: film_file_order_key проверяется сразу для каждой строки.
UPDATE film_file SET sort_order = sort_order + 20 WHERE file_role = 'poster';

INSERT INTO film_file (film_id, file_id, file_role, sort_order)
SELECT f.id, fl.id, 'poster', 1
  FROM (VALUES ('Побег из Шоушенка', '0b76b2a2-d1c7-4f04-a284-80ff7bb709a4'),
               ('Дюна: Часть третья', '556cc7b7-d4cc-4b12-ada6-126238fdfd8a'),
               ('Во все тяжкие', 'fb35416f-3b0d-4b96-bc65-cf6923f9e329'),
               ('Зелёная миля', '4057c4b8-8208-4a04-b169-26b0661453e3'),
               ('Форрест Гамп', '3560b757-9b95-45ec-af8c-623972370f9d'),
               ('Интерстеллар', '430042eb-ee69-4818-aed0-a312400a26bf'),
               ('Начало', '8ab9a119-dd74-44f0-baec-0629797483d7'),
               ('Остров проклятых', '617303b7-cfa7-4273-bd1d-63974bf68927'),
               ('Король Лев', '60aa1abc-b754-4817-ad9c-0bcda427a12b'),
               ('Один дома', '022a58e3-5b9b-411b-bfb3-09fedb700401'),
               ('Иван Васильевич меняет профессию', 'a7ef44b8-1983-4992-a889-da6f87a3f559'),
               ('Джентльмены', '637271d5-61b4-4e46-ac83-6d07494c7645'),
               ('Шерлок', 'f28c1ea2-47b0-49d5-b11c-9608744f0233'))
           AS v (title, image_id)
       JOIN film AS f ON f.title = v.title
       JOIN file AS fl ON fl.external_url LIKE '%/' || v.image_id || '/600x900';

-- У подборки «Фильмы о свободе» обложка ссылалась на несуществующий файл:
-- обложкой становится постер «Побега из Шоушенка».
UPDATE collection
   SET cover_file_id = (SELECT id FROM file
                         WHERE external_url LIKE '%/0b76b2a2-d1c7-4f04-a284-80ff7bb709a4/600x900')
 WHERE slug = 'films-about-freedom';
