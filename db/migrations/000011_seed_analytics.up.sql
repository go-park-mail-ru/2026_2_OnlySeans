INSERT INTO event_type (code, title)
VALUES ('page_view', 'Просмотр страницы'),
       ('click', 'Клик по элементу'),
       ('search', 'Поиск'),
       ('film_open', 'Открытие карточки фильма'),
       ('trailer_play', 'Запуск трейлера'),
       ('folder_add', 'Добавление в папку'),
       ('rating_set', 'Выставление оценки');

INSERT INTO page (path, title)
VALUES ('/', 'Главная'),
       ('/film/:id', 'Страница фильма'),
       ('/person/:id', 'Страница персоны'),
       ('/collection/:slug', 'Подборка'),
       ('/search', 'Результаты поиска'),
       ('/profile', 'Профиль пользователя');

INSERT INTO traffic_source (utm_source, utm_medium, utm_campaign)
VALUES ('vk', 'cpc', 'autumn_premieres'),
       ('yandex', 'cpc', 'brand'),
       ('telegram', 'social', NULL),
       ('direct', NULL, NULL);

INSERT INTO visitor (public_id, first_seen_at)
VALUES ('vis_8fJ2kQpL4nR7xW0a', now() - interval '20 days'),
       ('vis_Zq5TbN1cYv93MdLs', now() - interval '3 days');

INSERT INTO visit (visitor_id, account_id, traffic_source_id, started_at,
                   finished_at, user_agent, ip_address)
SELECT v.id, a.id, t.id, now() - interval '3 days',
       now() - interval '3 days' + interval '14 minutes',
       'Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/141.0', '203.0.113.17'
  FROM visitor AS v, account AS a, traffic_source AS t
 WHERE v.public_id = 'vis_8fJ2kQpL4nR7xW0a'
   AND a.username = 'anna'
   AND t.utm_source = 'vk';

INSERT INTO visit (visitor_id, traffic_source_id, started_at, finished_at,
                   user_agent, ip_address)
SELECT v.id, t.id, now() - interval '2 hours',
       now() - interval '2 hours' + interval '6 minutes',
       'Mozilla/5.0 (iPhone; CPU iPhone OS 18_2) Safari/605.1', '198.51.100.42'
  FROM visitor AS v, traffic_source AS t
 WHERE v.public_id = 'vis_Zq5TbN1cYv93MdLs'
   AND t.utm_source = 'telegram';

INSERT INTO analytic_event (visit_id, event_type_id, page_id, occurred_at)
SELECT vi.id, et.id, p.id, vi.started_at + interval '2 seconds'
  FROM visit AS vi, event_type AS et, page AS p
 WHERE vi.account_id IS NOT NULL
   AND et.code = 'page_view'
   AND p.path = '/';

INSERT INTO analytic_event (visit_id, event_type_id, page_id, film_id, occurred_at)
SELECT vi.id, et.id, p.id, f.id, vi.started_at + interval '45 seconds'
  FROM visit AS vi, event_type AS et, page AS p, film AS f
 WHERE vi.account_id IS NOT NULL
   AND et.code = 'film_open'
   AND p.path = '/film/:id'
   AND f.original_title = 'The Shawshank Redemption';

INSERT INTO analytic_event (visit_id, event_type_id, page_id, film_id,
                            element_code, occurred_at)
SELECT vi.id, et.id, p.id, f.id, 'trailer_play_button',
       vi.started_at + interval '70 seconds'
  FROM visit AS vi, event_type AS et, page AS p, film AS f
 WHERE vi.account_id IS NOT NULL
   AND et.code = 'click'
   AND p.path = '/film/:id'
   AND f.original_title = 'The Shawshank Redemption';

INSERT INTO analytic_event (visit_id, event_type_id, page_id, film_id,
                            element_code, occurred_at)
SELECT vi.id, et.id, p.id, f.id, 'add_to_favorite',
       vi.started_at + interval '3 minutes'
  FROM visit AS vi, event_type AS et, page AS p, film AS f
 WHERE vi.account_id IS NOT NULL
   AND et.code = 'folder_add'
   AND p.path = '/film/:id'
   AND f.original_title = 'The Shawshank Redemption';

INSERT INTO analytic_event (visit_id, event_type_id, page_id, occurred_at)
SELECT vi.id, et.id, p.id, vi.started_at + interval '12 seconds'
  FROM visit AS vi, event_type AS et, page AS p
 WHERE vi.account_id IS NULL
   AND et.code = 'search'
   AND p.path = '/search';

INSERT INTO search_query (visit_id, query_text, result_count)
SELECT vi.id, 'побег из шоушенка', 1
  FROM visit AS vi
 WHERE vi.account_id IS NULL;

INSERT INTO search_query (visit_id, query_text, result_count)
SELECT vi.id, 'фильмы про космос', 0
  FROM visit AS vi
 WHERE vi.account_id IS NULL;

INSERT INTO film_rating_daily (film_id, stat_date, rating_count, rating_sum,
                               average_score)
SELECT f.id, v.stat_date, v.rating_count, v.rating_sum,
       round(v.rating_sum::numeric / v.rating_count, 2)
  FROM film AS f,
       (VALUES (CURRENT_DATE - 2, 2, 19),
               (CURRENT_DATE - 1, 2, 20)) AS v (stat_date, rating_count, rating_sum)
 WHERE f.original_title = 'The Shawshank Redemption';
