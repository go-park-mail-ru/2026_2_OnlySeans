# Отношения и функциональные зависимости

Версия 4. 31 отношение в PostgreSQL 17 и 5 структур в Redis.
Типы и ограничения — в [fields.md](fields.md), ER-диаграммы в нотации Чена — в
[er.md](er.md).
Как схема пришла к этому виду по шагам нормальных форм — в
[normalization.md](normalization.md).

Обозначения: **PK** — первичный ключ, **UQ** — потенциальный (уникальный) ключ,
**FK** — внешний ключ.

Отношение «пользователь» называется `account`: `user` — зарезервированное слово
PostgreSQL, а `users` нарушало бы требование ДЗ о единственном числе в названиях
таблиц. `account` — собирательное название, оно разрешено.

---

## 1. Права доступа

### role
Роль пользователя: читатель, модератор, редактор каталога, администратор.

Атрибуты: id (PK), code (UQ), title (UQ), created_at, updated_at.

```
role:
{id} -> code, title, created_at, updated_at
{code} -> id, title, created_at, updated_at
{title} -> id, code, created_at, updated_at
```

### permission
Отдельное право: написать рецензию, модерировать контент, править каталог,
загрузить файл, управлять учётными записями.

Атрибуты: id (PK), code (UQ), title (UQ), created_at, updated_at.

```
permission:
{id} -> code, title, created_at, updated_at
{code} -> id, title, created_at, updated_at
{title} -> id, code, created_at, updated_at
```

### role_permission
Какие права даёт роль. Набор прав настраивается данными, а не кодом.

Атрибуты: role_id (PK, FK), permission_id (PK, FK), created_at.

```
role_permission:
{role_id, permission_id} -> created_at
```

### account
Учётная запись и профиль. Удаление мягкое: заполняется `deleted_at`, а
персональные данные стираются функцией `anonymize_account()`.

Атрибуты: id (PK), role_id (FK), email (UQ), username (UQ), password_hash,
first_name, gender, birth_date, bio, is_private, avatar_file_id (FK),
deleted_at, created_at, updated_at.

```
account:
{id} -> role_id, email, username, password_hash, first_name, gender, birth_date,
        bio, is_private, avatar_file_id, deleted_at, created_at, updated_at
{email} -> id, role_id, username, password_hash, first_name, gender, birth_date,
           bio, is_private, avatar_file_id, deleted_at, created_at, updated_at
{username} -> id, role_id, email, password_hash, first_name, gender, birth_date,
              bio, is_private, avatar_file_id, deleted_at, created_at, updated_at
```

---

## 2. Файлы

### file
Файл продукта: аватар, постер, кадр, трейлер, обложка подборки. У файла есть
собственные служебные поля, поэтому он вынесен в отдельное отношение, а не
хранится ссылкой в карточке фильма.

Атрибуты: id (PK), storage_key (UQ), external_url (UQ), mime_type, size_bytes,
created_by (FK), created_at, updated_at.

```
file:
{id} -> storage_key, external_url, mime_type, size_bytes, created_by,
        created_at, updated_at
{storage_key} -> id, external_url, mime_type, size_bytes, created_by,
                 created_at, updated_at
{external_url} -> id, storage_key, mime_type, size_bytes, created_by,
                  created_at, updated_at
```

### film_file
Файлы фильма с ролью и порядком показа. Именно это отношение позволяет иметь у
фильма несколько постеров и несколько трейлеров.

Атрибуты: film_id (PK, FK), file_id (PK, FK), file_role, sort_order, created_at.
UQ: (film_id, file_role, sort_order).

```
film_file:
{film_id, file_id} -> file_role, sort_order, created_at
{film_id, file_role, sort_order} -> file_id, created_at
```

---

## 3. Каталог

### film
Карточка фильма или сериала: и то и другое лежит в одном отношении, потому что у
них общие страницы, жанры, актёры, оценки и рецензии. Тип задаёт `film_type`.

Атрибуты: id (PK), title, original_title, film_type, production_year,
duration_min, age_limit, description, created_at, updated_at.
UQ: (id, film_type).

```
film:
{id} -> title, original_title, film_type, production_year, duration_min,
        age_limit, description, created_at, updated_at
{id, film_type} -> title, original_title, production_year, duration_min,
                   age_limit, description, created_at, updated_at
```
Вторая зависимость — тривиальное следствие первой: `film_type` определяется
идентификатором. Уникальный ключ (id, film_type) объявлен не ради неё, а чтобы
на эту пару мог сослаться составной внешний ключ отношения `season`.

`production_year` — год производства, а не дата премьеры: даты выхода лежат в
`film_release`, поэтому дублирования нет. Средняя оценка фильма не хранится:
это производное значение от `film_rating`.

### genre
Справочник жанров. Атрибуты: id (PK), name (UQ), slug (UQ), created_at, updated_at.

```
genre:
{id} -> name, slug, created_at, updated_at
{name} -> id, slug, created_at, updated_at
{slug} -> id, name, created_at, updated_at
```

### film_genre
Связь фильма и жанра, многие ко многим.
Атрибуты: film_id (PK, FK), genre_id (PK, FK), created_at.

```
film_genre:
{film_id, genre_id} -> created_at
```

### country
Справочник стран. Атрибуты: id (PK), name (UQ), iso_code (UQ), created_at, updated_at.

```
country:
{id} -> name, iso_code, created_at, updated_at
{name} -> id, iso_code, created_at, updated_at
{iso_code} -> id, name, created_at, updated_at
```

### film_country
Страны производства фильма.
Атрибуты: film_id (PK, FK), country_id (PK, FK), created_at.

```
film_country:
{film_id, country_id} -> created_at
```

### similar_film
Похожие фильмы. Пара хранится один раз в возрастающем порядке идентификаторов,
это обеспечивает CHECK `film_id < similar_film_id`: похожесть симметрична, и
строки (5, 9) и (9, 5) одновременно существовать не могут.

Атрибуты: film_id (PK, FK), similar_film_id (PK, FK), created_at.

```
similar_film:
{film_id, similar_film_id} -> created_at
```

### film_release
Дата премьеры фильма в стране для конкретного типа проката. Дата входит в
первичный ключ, поэтому повторный прокат возможен.

Атрибуты: film_id (PK, FK), country_id (PK, FK), release_type (PK),
release_date (PK), created_at, updated_at.

```
film_release:
{film_id, country_id, release_type, release_date} -> created_at, updated_at
```

### season
Сезон сериала. Ссылается на пару (film_id, film_type) отношения `film`, а
`film_type` здесь закреплён значением `'series'`, поэтому сезон невозможно
завести у фильма.

Атрибуты: film_id (PK, FK), film_type (FK), season_number (PK), title,
created_at, updated_at.

```
season:
{film_id, season_number} -> film_type, title, created_at, updated_at
```
`film_type` в левой части не нужен: он константа, равная `'series'` для всех
строк отношения.

### episode
Серия сезона. Ссылается на сезон составным внешним ключом (film_id, season_number).

Атрибуты: film_id (PK, FK), season_number (PK, FK), episode_number (PK), title,
duration_min, release_date, created_at, updated_at.

```
episode:
{film_id, season_number, episode_number} -> title, duration_min, release_date,
                                            created_at, updated_at
```

---

## 4. Персоны

### person
Персона: актёр, режиссёр, сценарист. Профессия хранится не здесь, а в
`film_person`.

Атрибуты: id (PK), first_name, last_name, birth_date, photo_file_id (FK),
created_at, updated_at.

```
person:
{id} -> first_name, last_name, birth_date, photo_file_id, created_at, updated_at
```
Тёзки и однофамильцы не мешают: ключ суррогатный, а имя и фамилия не объявлены
уникальными.

### role_type
Справочник кинопрофессий.
Атрибуты: id (PK), code (UQ), title (UQ), created_at, updated_at.

```
role_type:
{id} -> code, title, created_at, updated_at
{code} -> id, title, created_at, updated_at
{title} -> id, code, created_at, updated_at
```

### film_person
Участие персоны в фильме в определённой профессии.

Атрибуты: id (PK), film_id (FK), person_id (FK), role_type_id (FK),
character_name, created_at, updated_at.
UQ: (film_id, person_id, role_type_id, character_name).

```
film_person:
{id} -> film_id, person_id, role_type_id, character_name, created_at, updated_at
{film_id, person_id, role_type_id, character_name} -> id, created_at, updated_at
```
`character_name` зависит от всей комбинации «фильм + персона + профессия», а не
от персоны, поэтому находится именно здесь.

Один человек может участвовать в фильме в нескольких профессиях: сценарист,
который снялся в эпизоде, даёт две строки, различающиеся `role_type_id`. Один
актёр, сыгравший двух персонажей, даёт две строки, различающиеся
`character_name`. Совпадение имени персонажа с именем самой персоны схеме
безразлично: это несвязанные атрибуты разных отношений.

---

## 5. Оценки и рецензии

### film_rating
Оценка фильма пользователем от 1 до 10.
Атрибуты: account_id (PK, FK), film_id (PK, FK), score, created_at, updated_at.

```
film_rating:
{account_id, film_id} -> score, created_at, updated_at
```

### film_rating_history
История изменения оценок. Заполняется триггером, строки не изменяются.
Внешний ключ на `account` — RESTRICT, поэтому история переживает удаление
профиля (оно мягкое).

Атрибуты: id (PK), account_id (FK), film_id (FK), old_score, new_score, created_at.

```
film_rating_history:
{id} -> account_id, film_id, old_score, new_score, created_at
```

### review
Рецензия пользователя на фильм, одна на пару «пользователь + фильм». Состояние
проверки хранится не здесь, а в `moderation_task`.

Атрибуты: id (PK), account_id (FK), film_id (FK), title, content,
contains_spoilers, deleted_at, deleted_by (FK), created_at, updated_at.
UQ: (account_id, film_id).

```
review:
{id} -> account_id, film_id, title, content, contains_spoilers, deleted_at,
        deleted_by, created_at, updated_at
{account_id, film_id} -> id, title, content, contains_spoilers, deleted_at,
                         deleted_by, created_at, updated_at
```

---

## 6. Папки, подписки, подборки

### folder
Папка пользователя. Системная папка каждого типа одна, это обеспечивает
частичный уникальный индекс.

Атрибуты: id (PK), account_id (FK), title, folder_type, is_private, created_at,
updated_at. UQ: (account_id, title).

```
folder:
{id} -> account_id, title, folder_type, is_private, created_at, updated_at
{account_id, title} -> id, folder_type, is_private, created_at, updated_at
```

### folder_item
Элемент папки: фильм или персона. Обе ссылки — настоящие внешние ключи, выбор
между ними задаёт CHECK `num_nonnulls(film_id, person_id) = 1`.

Атрибуты: id (PK), folder_id (FK), film_id (FK), person_id (FK), created_at.
UQ: (folder_id, film_id), (folder_id, person_id).

```
folder_item:
{id} -> folder_id, film_id, person_id, created_at
{folder_id, film_id} -> id, person_id, created_at
{folder_id, person_id} -> id, film_id, created_at
```

### release_subscription
Подписка на выход фильма.
Атрибуты: account_id (PK, FK), film_id (PK, FK), created_at.

```
release_subscription:
{account_id, film_id} -> created_at
```

### collection
Подборка фильмов.
Атрибуты: id (PK), title, slug (UQ), description, cover_file_id (FK),
created_at, updated_at.

```
collection:
{id} -> title, slug, description, cover_file_id, created_at, updated_at
{slug} -> id, title, description, cover_file_id, created_at, updated_at
```

### collection_film
Фильмы в подборке с порядком показа.
Атрибуты: collection_id (PK, FK), film_id (PK, FK), position, created_at.
UQ: (collection_id, position).

```
collection_film:
{collection_id, film_id} -> position, created_at
{collection_id, position} -> film_id, created_at
```

---

## 7. Уведомления, чат, модерация

### notification
Уведомление пользователю, доставляется по вебсокету.
Атрибуты: id (PK), account_id (FK), message, is_read, created_at, updated_at.

```
notification:
{id} -> account_id, message, is_read, created_at, updated_at
```

### film_discussion
Комната обсуждения фильма, одна на фильм.
Атрибуты: id (PK), film_id (UQ, FK), is_closed, created_at, updated_at.

```
film_discussion:
{id} -> film_id, is_closed, created_at, updated_at
{film_id} -> id, is_closed, created_at, updated_at
```

### discussion_message
Сообщение в обсуждении. Поля `deleted_at` и `deleted_by` показывают, кто удалил
сообщение: автор или модератор.

Атрибуты: id (PK), discussion_id (FK), account_id (FK), message_text,
deleted_at, deleted_by (FK), created_at, updated_at.

```
discussion_message:
{id} -> discussion_id, account_id, message_text, deleted_at, deleted_by,
        created_at, updated_at
```

### account_ban
Блокировка пользователя на период -- крайняя мера модерации. Ограничение EXCLUDE
запрещает пересекающиеся блокировки одного пользователя, поэтому на вопрос
«какая блокировка действует сейчас» всегда есть ровно один ответ.

Атрибуты: id (PK), account_id (FK), moderator_id (FK), reason, active_during,
created_at, updated_at.

```
account_ban:
{id} -> account_id, moderator_id, reason, active_during, created_at, updated_at
{account_id, active_during} -> id, moderator_id, reason, created_at, updated_at
```
Вторая зависимость обеспечена ограничением EXCLUDE: периоды блокировок одного
пользователя не пересекаются, поэтому пара «пользователь + период» уникальна.

### moderation_task
Проверка рецензии или сообщения: автоматическая (нейросеть) или ручная.
Публикуется то, что получило `approved`.

Атрибуты: id (PK), review_id (FK), message_id (FK), status, source, model_name,
model_score, decided_by (FK), reason, decided_at, created_at, updated_at.

```
moderation_task:
{id} -> review_id, message_id, status, source, model_name, model_score,
        decided_by, reason, decided_at, created_at, updated_at
```

---

## 8. Отношения вне PostgreSQL

### session, email_verification, password_reset (Redis)
```
session:
{session_id} -> account_id

email_verification:
{token} -> account_id

password_reset:
{token} -> account_id
```
Срок жизни задан TTL: 30 дней, 24 часа и 15 минут соответственно, поэтому поля
с датой окончания не нужно.

### rate_limit (Redis)
```
rate_limit:
{ip_address} -> attempt_count
```
Счётчик попыток входа, TTL 15 минут.

### film_top_cache (Redis)
Кэш топа фильмов, TTL 1 час. В виде отношения не представим: это сериализованный
список, который перезаписывается целиком. Исходные данные остаются в
`film_rating`.

### S3 / MinIO
Файлы лежат в бакетах объектного хранилища (бакет — контейнер верхнего уровня со
своими правами доступа), в PostgreSQL хранится только ключ объекта или внешний
адрес — в отношении `file`. Бакеты: `avatar`, `poster`, `still`, `person`, `cover`.

---

## 9. Доказательство нормальных форм

### Первая нормальная форма
Все атрибуты атомарны, составных типов (массивы, JSON) нет. Диапазон
`tstzrange` в `account_ban` -- скалярное значение встроенного типа: его нельзя
разложить на переменное число элементов, он сравнивается и индексируется
целиком, а пара полей «начало» и «конец» не позволила бы наложить EXCLUDE. Жанры, страны,
файлы, участники фильма, элементы папок и фильмы в подборках вынесены в
отдельные отношения. Поля `poster_url` и `trailer_url`, которые раньше
подразумевали ровно один файл, заменены отношением `film_file`. Повторяющихся
групп вида `poster_1`, `poster_2` в схеме нет.

### Вторая нормальная форма
Схема в 1НФ, и каждый неключевой атрибут функционально полно зависит от всего
первичного ключа. Проверка касается отношений с составным ключом:

- **`film_release`** — отношение всеключевое: все четыре атрибута
  (film_id, country_id, release_type, release_date) входят в первичный ключ.
  Неключевых атрибутов, кроме служебных меток времени, нет, поэтому частичная
  зависимость невозможна по определению. Метки времени относятся к строке
  целиком: это момент появления и правки именно этой записи о премьере. Дату
  премьеры нельзя держать полем в `film`, потому что она определяется
  сочетанием фильма, страны и типа проката;
- `film_rating`: `score` зависит от пары «пользователь + фильм», по отдельности
  ни от того, ни от другого;
- `season`: `title` зависит от пары «фильм + номер сезона»; `film_type` —
  константа `'series'`, существующая ради внешнего ключа;
- `episode`: `title`, `duration_min`, `release_date` зависят от тройки
  «фильм + сезон + номер серии»;
- `collection_film`: `position` зависит от пары «подборка + фильм»;
- `film_file`: `file_role` и `sort_order` зависят от пары «фильм + файл»;
- `film_person`: ключ суррогатный, но и по потенциальному ключу частичных
  зависимостей нет;
- в остальных связующих отношениях (`film_genre`, `film_country`,
  `similar_film`, `role_permission`, `release_subscription`) неключевых
  атрибутов, кроме `created_at`, нет.

### Третья нормальная форма
Схема во 2НФ, транзитивных зависимостей неключевых атрибутов друг от друга нет.
Справочные данные вынесены:

- название и slug жанра — в `genre`, код страны — в `country`;
- название профессии — в `role_type`, а не в `film_person`;
- имя и фото персоны — в `person`;
- тип, размер и адрес файла — в `file`, а `film_file` хранит только роль файла
  в конкретном фильме;
- права роли — в `role_permission`, а не текстом в `account`.

Производные значения не хранятся: средняя оценка фильма, средняя длительность
серии сериала, количество рецензий, возраст персоны вычисляются запросами.

### Нормальная форма Бойса-Кодда
Для каждого отношения выписаны все ФЗ, и в левой части каждой стоит
потенциальный ключ:

- в отношениях с суррогатным ключом детерминанты — `{id}` и уникальные атрибуты
  (`email`, `username`, `code`, `title`, `slug`, `storage_key`, `external_url`,
  `film_id` в `film_discussion`), каждый объявлен ограничением UNIQUE;
- в связующих отношениях единственный детерминант — составной первичный ключ;
- в отношениях с естественным составным ключом (`film_release`, `film_rating`,
  `season`, `episode`, `collection_film`, `film_file`, `folder_item`) все
  детерминанты объявлены как PRIMARY KEY или UNIQUE;
- в `account_ban` второй детерминант (account_id, active_during) обеспечен
  ограничением EXCLUDE, которое здесь играет роль уникального ключа.

Ситуации, когда неключевой атрибут определяет другой атрибут, в схеме нет,
поэтому все отношения находятся в НФБК.
