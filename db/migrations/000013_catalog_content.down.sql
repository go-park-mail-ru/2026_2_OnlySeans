DELETE FROM collection
 WHERE slug IN ('best-of-all-time', 'family', 'mind-benders', 'weekend-series');

DELETE FROM film_file
 WHERE file_id IN (SELECT id FROM file WHERE storage_key LIKE 'posters/%');

UPDATE film_file
   SET sort_order = sort_order - 10
 WHERE file_role = 'poster'
   AND sort_order > 10
   AND film_id IN (SELECT id FROM film
                    WHERE original_title IN ('The Shawshank Redemption', 'Breaking Bad'));

DELETE FROM file WHERE storage_key LIKE 'posters/%';

DELETE FROM film
 WHERE title IN ('Зелёная миля', 'Форрест Гамп', 'Интерстеллар', 'Начало',
                 'Остров проклятых', 'Король Лев', 'Один дома',
                 'Иван Васильевич меняет профессию', 'Джентльмены', 'Шерлок');

DELETE FROM genre WHERE slug IN ('animation', 'adventure', 'family');
