BEGIN;

CREATE TEMP TABLE test_result (
    step   integer,
    name   text,
    passed boolean,
    detail text
) ON COMMIT DROP;

CREATE OR REPLACE FUNCTION pg_temp.expect(
    step integer, name text, statement text, should_fail boolean
) RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    failed  boolean := false;
    message text := '';
BEGIN
    BEGIN
        EXECUTE statement;
    EXCEPTION WHEN others THEN
        failed := true;
        message := sqlstate || ' ' || sqlerrm;
    END;

    INSERT INTO test_result
    VALUES (step, name, failed = should_fail,
            CASE WHEN failed THEN message ELSE 'выполнено без ошибки' END);
END;
$$;

SELECT pg_temp.expect(1, 'сезон у фильма запрещён',
    format('INSERT INTO season (film_id, season_number) VALUES (%s, 1)',
           (SELECT id FROM film WHERE film_type = 'movie' LIMIT 1)), true);

SELECT pg_temp.expect(2, 'duration_min у сериала запрещён',
    format('UPDATE film SET duration_min = 45 WHERE id = %s',
           (SELECT id FROM film WHERE film_type = 'series' LIMIT 1)), true);

SELECT pg_temp.expect(3, 'обратная пара в similar_film запрещена',
    format('INSERT INTO similar_film (film_id, similar_film_id) VALUES (%s, %s)',
           (SELECT max(similar_film_id) FROM similar_film),
           (SELECT min(film_id) FROM similar_film)), true);

SELECT pg_temp.expect(4, 'повторный прокат разрешён',
    format($sql$INSERT INTO film_release (film_id, country_id, release_type, release_date)
                VALUES (%s, %s, 'cinema', DATE '2030-01-01')$sql$,
           (SELECT film_id FROM film_release LIMIT 1),
           (SELECT country_id FROM film_release LIMIT 1)), false);

SELECT pg_temp.expect(5, 'два постера с одним sort_order запрещены',
    format($sql$INSERT INTO film_file (film_id, file_id, file_role, sort_order)
                VALUES (%s, %s, 'poster', 1)$sql$,
           (SELECT film_id FROM film_file WHERE file_role = 'poster' LIMIT 1),
           (SELECT id FROM file WHERE storage_key = 'avatar/anna.png')), true);

SELECT pg_temp.expect(6, 'удаление автора рецензии запрещено',
    format('DELETE FROM account WHERE id = %s',
           (SELECT account_id FROM review LIMIT 1)), true);

SELECT pg_temp.expect(7, 'anonymize_account выполняется',
    format('SELECT anonymize_account(%s)',
           (SELECT account_id FROM review LIMIT 1)), false);

INSERT INTO test_result
SELECT 8, 'рецензии и история пережили анонимизацию',
       (SELECT count(*) FROM review) > 0
       AND (SELECT count(*) FROM film_rating_history) > 0,
       format('рецензий %s, строк истории %s',
              (SELECT count(*) FROM review),
              (SELECT count(*) FROM film_rating_history));

INSERT INTO test_result
SELECT 9, 'персональные данные стёрты',
       count(*) = 0,
       format('записей с остатками профиля: %s', count(*))
  FROM account
 WHERE deleted_at IS NOT NULL
   AND (first_name IS NOT NULL OR bio IS NOT NULL OR birth_date IS NOT NULL
        OR gender IS NOT NULL OR avatar_file_id IS NOT NULL);

DO $$
DECLARE
    before_count integer;
    after_count  integer;
    target       record;
BEGIN
    SELECT count(*) INTO before_count FROM film_rating_history;
    SELECT account_id, film_id, score INTO target FROM film_rating LIMIT 1;

    UPDATE film_rating
       SET score = CASE WHEN target.score = 1 THEN 2 ELSE target.score - 1 END
     WHERE account_id = target.account_id AND film_id = target.film_id;

    SELECT count(*) INTO after_count FROM film_rating_history;

    INSERT INTO test_result
    VALUES (10, 'триггер истории оценок работает', after_count = before_count + 1,
            format('было %s, стало %s', before_count, after_count));
END;
$$;

SELECT pg_temp.expect(11, 'folder_item с двумя ссылками запрещён',
    format($sql$INSERT INTO folder_item (folder_id, film_id, person_id)
                VALUES (%s, %s, %s)$sql$,
           (SELECT id FROM folder LIMIT 1),
           (SELECT id FROM film LIMIT 1),
           (SELECT id FROM person LIMIT 1)), true);

SELECT pg_temp.expect(12, 'пустой folder_item запрещён',
    format('INSERT INTO folder_item (folder_id) VALUES (%s)',
           (SELECT id FROM folder LIMIT 1)), true);

SELECT pg_temp.expect(13, 'вторая задача модерации в очереди запрещена',
    format($sql$INSERT INTO moderation_task (message_id, status, source, model_name)
                VALUES (%s, 'pending', 'auto', 'toxicity-ru-v1')$sql$,
           (SELECT message_id FROM moderation_task
             WHERE message_id IS NOT NULL AND status = 'pending' LIMIT 1)), true);

SELECT pg_temp.expect(14, 'auto-проверка без model_name запрещена',
    format($sql$INSERT INTO moderation_task (review_id, status, source)
                VALUES (%s, 'pending', 'auto')$sql$,
           (SELECT id FROM review LIMIT 1)), true);

SELECT pg_temp.expect(15, 'ручное решение без модератора запрещено',
    format($sql$INSERT INTO moderation_task (review_id, status, source, decided_at)
                VALUES (%s, 'approved', 'manual', now())$sql$,
           (SELECT id FROM review LIMIT 1)), true);

SELECT pg_temp.expect(16, 'вторая системная папка запрещена',
    format($sql$INSERT INTO folder (account_id, title, folder_type)
                VALUES (%s, 'Ещё избранное', 'favorite')$sql$,
           (SELECT account_id FROM folder WHERE folder_type = 'favorite' LIMIT 1)), true);

SELECT pg_temp.expect(17, 'оценка 11 запрещена',
    format($sql$INSERT INTO film_rating (account_id, film_id, score)
                VALUES (%s, %s, 11)$sql$,
           (SELECT id FROM account ORDER BY id DESC LIMIT 1),
           (SELECT id FROM film ORDER BY id DESC LIMIT 1)), true);

SELECT pg_temp.expect(18, 'файл с двумя адресами запрещён',
    $sql$INSERT INTO file (storage_key, external_url, mime_type, size_bytes)
         VALUES ('poster/x.jpg', 'https://example.com/x.jpg', 'image/jpeg', 100)$sql$,
    true);

SELECT pg_temp.expect(19, 'внешний адрес по grpc допустим',
    $sql$INSERT INTO file (external_url, mime_type)
         VALUES ('grpc://media.internal/poster/42', 'image/jpeg')$sql$,
    false);

DO $$
DECLARE
    f_id bigint;
    p_id bigint;
    ok   boolean := true;
    msg  text := 'обе роли добавлены';
BEGIN
    SELECT id INTO f_id FROM film WHERE film_type = 'movie' LIMIT 1;

    INSERT INTO person (first_name, last_name) VALUES ('Стэн', 'Ли')
    RETURNING id INTO p_id;

    INSERT INTO film_person (film_id, person_id, role_type_id, character_name)
    SELECT f_id, p_id, id, NULL FROM role_type WHERE code = 'writer';

    INSERT INTO film_person (film_id, person_id, role_type_id, character_name)
    SELECT f_id, p_id, id, 'Стэн Ли' FROM role_type WHERE code = 'actor';

    INSERT INTO test_result VALUES (20, 'сценарист снялся в своём фильме', ok, msg);
EXCEPTION WHEN others THEN
    INSERT INTO test_result
    VALUES (20, 'сценарист снялся в своём фильме', false, sqlstate || ' ' || sqlerrm);
END;
$$;

SELECT pg_temp.expect(21, 'повторная одинаковая роль запрещена',
    format($sql$INSERT INTO film_person (film_id, person_id, role_type_id, character_name)
                SELECT film_id, person_id, role_type_id, character_name
                  FROM film_person WHERE id = %s$sql$,
           (SELECT id FROM film_person LIMIT 1)), true);

SELECT pg_temp.expect(22, 'deleted_at без deleted_by запрещён',
    format('UPDATE review SET deleted_at = now() WHERE id = %s',
           (SELECT id FROM review LIMIT 1)), true);

SELECT pg_temp.expect(23, 'пересекающиеся блокировки запрещены',
    format($sql$INSERT INTO account_ban (account_id, moderator_id, reason, active_during)
                VALUES (%s, %s, 'Повторное нарушение',
                        tstzrange(now() + interval '1 day', now() + interval '30 days'))$sql$,
           (SELECT account_id FROM account_ban LIMIT 1),
           (SELECT moderator_id FROM account_ban LIMIT 1)), true);

SELECT pg_temp.expect(24, 'блокировка после окончания прежней разрешена',
    format($sql$INSERT INTO account_ban (account_id, moderator_id, reason, active_during)
                VALUES (%s, %s, 'Новое нарушение',
                        tstzrange(now() + interval '60 days', now() + interval '90 days'))$sql$,
           (SELECT account_id FROM account_ban LIMIT 1),
           (SELECT moderator_id FROM account_ban LIMIT 1)), false);

SELECT pg_temp.expect(25, 'самоблокировка запрещена',
    format($sql$INSERT INTO account_ban (account_id, moderator_id, reason, active_during)
                VALUES (%s, %s, 'Проверка', tstzrange(now(), now() + interval '1 day'))$sql$,
           (SELECT moderator_id FROM account_ban LIMIT 1),
           (SELECT moderator_id FROM account_ban LIMIT 1)), true);

SELECT pg_temp.expect(26, 'правка фильма пишет версию',
    format('UPDATE film SET title = title || %L WHERE id = %s', ' (ред.)',
           (SELECT id FROM film LIMIT 1)), false);

INSERT INTO test_result
SELECT 27, 'версия карточки сохранена', count(*) > 0,
       format('строк в film_revision: %s', count(*))
  FROM film_revision;

SELECT pg_temp.expect(28, 'правка рецензии пишет версию',
    format('UPDATE review SET content = content || %L WHERE id = %s', ' Дополнено.',
           (SELECT id FROM review LIMIT 1)), false);

INSERT INTO test_result
SELECT 29, 'версия рецензии сохранена', count(*) > 0,
       format('строк в review_revision: %s', count(*))
  FROM review_revision;

INSERT INTO test_result
SELECT 30, 'история ролей заполнена при регистрации', count(*) >= 5,
       format('строк в account_role_history: %s', count(*))
  FROM account_role_history;

SELECT pg_temp.expect(31, 'смена роли пишет историю',
    format('UPDATE account SET role_id = %s WHERE id = %s',
           (SELECT id FROM role WHERE code = 'moderator'),
           (SELECT id FROM account WHERE username = 'ivan')), false);

INSERT INTO test_result
SELECT 32, 'смена роли попала в историю', count(*) = 1,
       format('записей о смене роли: %s', count(*))
  FROM account_role_history
 WHERE old_role_id IS NOT NULL;

SELECT pg_temp.expect(33, 'событие с двумя объектами запрещено',
    format($sql$INSERT INTO analytic_event (visit_id, event_type_id, film_id,
                                            person_id, occurred_at)
                VALUES (%s, %s, %s, %s, now())$sql$,
           (SELECT id FROM visit LIMIT 1),
           (SELECT id FROM event_type WHERE code = 'click'),
           (SELECT id FROM film LIMIT 1),
           (SELECT id FROM person LIMIT 1)), true);

SELECT pg_temp.expect(34, 'событие без визита запрещено',
    format($sql$INSERT INTO analytic_event (visit_id, event_type_id, occurred_at)
                VALUES (0, %s, now())$sql$,
           (SELECT id FROM event_type WHERE code = 'click')), true);

SELECT pg_temp.expect(35, 'визит, завершённый раньше начала, запрещён',
    format($sql$INSERT INTO visit (visitor_id, started_at, finished_at)
                VALUES (%s, now(), now() - interval '1 hour')$sql$,
           (SELECT id FROM visitor LIMIT 1)), true);

SELECT pg_temp.expect(36, 'повторная utm-метка запрещена',
    $sql$INSERT INTO traffic_source (utm_source, utm_medium, utm_campaign)
         VALUES ('telegram', 'social', NULL)$sql$, true);

SELECT pg_temp.expect(37, 'два снимка рейтинга за один день запрещены',
    format($sql$INSERT INTO film_rating_daily (film_id, stat_date, rating_count,
                                               rating_sum, average_score)
                VALUES (%s, CURRENT_DATE - 1, 3, 27, 9.0)$sql$,
           (SELECT film_id FROM film_rating_daily LIMIT 1)), true);

SELECT pg_temp.expect(38, 'вторая действующая рецензия запрещена',
    format($sql$INSERT INTO review (account_id, film_id, title, content)
                VALUES (%s, %s, 'Ещё раз о том же',
                        'Второй отзыв того же пользователя на тот же фильм появиться не должен, пока первый не удалён.')$sql$,
           (SELECT account_id FROM review WHERE deleted_at IS NULL LIMIT 1),
           (SELECT film_id FROM review WHERE deleted_at IS NULL LIMIT 1)), true);

SELECT pg_temp.expect(39, 'удаление рецензии автором',
    format($sql$UPDATE review
                   SET deleted_at = now(),
                       deleted_by = account_id
                 WHERE id = %s$sql$,
           (SELECT id FROM review WHERE deleted_at IS NULL LIMIT 1)), false);

SELECT pg_temp.expect(40, 'после удаления можно написать новую рецензию',
    format($sql$INSERT INTO review (account_id, film_id, title, content)
                VALUES (%s, %s, 'Передумала',
                        'После удаления прежнего отзыва пользователь может написать новый на тот же фильм.')$sql$,
           (SELECT account_id FROM review WHERE deleted_at IS NOT NULL LIMIT 1),
           (SELECT film_id FROM review WHERE deleted_at IS NOT NULL LIMIT 1)), false);

INSERT INTO test_result
SELECT 41, 'удалённая рецензия осталась в таблице', count(*) = 2,
       format('рецензий на пару пользователь-фильм: %s (из них удалённых %s)',
              count(*), count(*) FILTER (WHERE deleted_at IS NOT NULL))
  FROM review
 WHERE (account_id, film_id) = (SELECT account_id, film_id
                                  FROM review
                                 WHERE deleted_at IS NOT NULL
                                 LIMIT 1);

SELECT step,
       name AS "проверка",
       CASE WHEN passed THEN 'OK' ELSE 'ОШИБКА' END AS "результат",
       detail AS "подробности"
  FROM test_result
 ORDER BY step;

SELECT count(*) FILTER (WHERE passed)       AS "прошло",
       count(*) FILTER (WHERE NOT passed)   AS "не прошло",
       count(*)                             AS "всего"
  FROM test_result;

ROLLBACK;
