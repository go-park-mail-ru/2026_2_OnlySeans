# Словарь данных: все поля, типы и ограничения

Версия 4. 31 таблица в PostgreSQL 17 и 5 структур в Redis.
Схема применена и проверена на живой базе: 25 проверок ограничений из 25 пройдены.
Дополняет [relations.md](relations.md): там отношения и функциональные
зависимости, здесь — типы и ограничения целостности с обоснованием.
ER-диаграммы — в [er.md](er.md), DDL целиком — в `db/migrations`.

Целевая версия СУБД — **PostgreSQL 17**.

Сокращения: **PK** — первичный ключ, **FK** — внешний ключ, **UQ** — уникальный
ключ, **NN** — NOT NULL.

---

## 0. Общие правила

| Решение | Почему так |
|---|---|
| `bigint GENERATED ALWAYS AS IDENTITY` | `serial` — устаревший синтаксис поверх последовательности, он не запрещает вставку значения вручную. IDENTITY описан в стандарте SQL |
| `text` + `CHECK (length(...))` вместо `varchar(n)` | работают одинаково, но границу у CHECK можно менять без перезаписи таблицы, и она задаётся с двух сторон |
| `timestamptz` вместо `timestamp` | `timestamp` не хранит часовой пояс, и момент времени становится неоднозначным |
| `date` для дат без времени | дата рождения, премьеры, выхода серии |
| `smallint` для небольших чисел | оценка 1–10, год, номера сезонов и серий, порядок сортировки |
| `numeric(4,3)` для оценки модели | дробное значение 0…1 с фиксированной точностью, без двоичной плавающей точки |
| Нет `array` и `json` | запрещено условием задания; любое множество вынесено в отдельное отношение |
| **Индексов нет** | создаются только те, что СУБД делает сама под PRIMARY KEY и UNIQUE. Три частичных уникальных индекса — не оптимизация, а ограничения целостности, которые обычным UNIQUE не выразить. Индексы для скорости будут отдельной работой, каждый со своим обоснованием |
| **Нет `ON UPDATE CASCADE`** | первичные ключи — IDENTITY, они никогда не меняются, поэтому каскад по UPDATE давал бы только лишние срабатывания |
| `ON DELETE RESTRICT` на пользовательский контент | рецензии, оценки, история оценок и сообщения чата не исчезают вместе с учётной записью. Удаление аккаунта — мягкое, через `anonymize_account()` |
| `ON DELETE CASCADE` на личные данные | папки, подписки, уведомления существуют только для своего владельца |
| `ON DELETE SET NULL` на файлы | если файл удалён, ссылка на аватар или обложку обнуляется |
| `deleted_at` + `deleted_by` вместо `is_deleted` | видно не только что объект удалён, но и когда и кем: автором или модератором |
| Файлы — отдельная таблица | у фильма может быть несколько постеров и трейлеров, а у каждого файла свои служебные поля: тип, размер, кто и когда загрузил |
| `created_at` и `updated_at` описаны отдельными строками | это разные поля с разным смыслом: первое ставится один раз при вставке, второе переписывает триггер при каждом изменении |
| Адрес файла не привязан к `https` | доступ к файлам может идти по http внутри контура, по gRPC или по внутреннему протоколу хранилища, поэтому проверяется только наличие схемы вида `протокол://`, а не конкретный протокол |

### Про имя таблицы `account`
`user` — зарезервированное слово PostgreSQL, писать его пришлось бы в кавычках.
Вариант `users` формой множественного числа нарушает требование ДЗ по СУБД
(«нельзя использовать множественное число в названиях таблиц»), поэтому взято
собирательное `account` — оно разрешено тем же требованием.

### Про слово «бакет»
Бакет — это именованный контейнер верхнего уровня в объектном хранилище
(S3/MinIO), аналог корневой папки: у него свои права доступа и политика
жизненного цикла. Ключ объекта внутри бакета (`file.storage_key`) — это полный
путь к файлу. В базе хранится только бакет с ключом, сам файл лежит в хранилище.

---

## 1. role — роль пользователя

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| code | text | NN, UQ, CHECK `^[a-z_]{3,32}$` | код для кода приложения: reader, moderator, editor, admin |
| title | text | NN, UQ, CHECK 2–64 | подпись для интерфейса |
| created_at | timestamptz | NN, DEFAULT now() | момент создания роли |
| updated_at | timestamptz | NN, DEFAULT now() | переписывается триггером при изменении строки |

## 2. permission — отдельное право

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| code | text | NN, UQ, CHECK `^[a-z_.]{3,64}$` | точка разделяет объект и действие: `review.moderate`, `file.upload` |
| title | text | NN, UQ, CHECK 2–128 | подпись для интерфейса |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 3. role_permission — права роли

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| role_id | bigint | PK, FK → role, CASCADE | |
| permission_id | bigint | PK, FK → permission, CASCADE | |
| created_at | timestamptz | NN, DEFAULT now() | `updated_at` нет: строка не изменяется, право либо выдано, либо нет |

Набор прав настраивается данными, а не кодом: чтобы дать модератору право
править каталог, достаточно вставить строку. Одного текстового поля `role`
для этого недостаточно.

## 4. account — учётная запись и профиль

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | суррогатный ключ: email и username пользователь может сменить |
| role_id | bigint | NN, FK → role, RESTRICT | роль нельзя удалить, пока она кому-то назначена |
| email | text | NN, UQ, CHECK маска и ≤ 254 | 254 — предел длины адреса по RFC 5321 |
| username | text | NN, UQ, CHECK `^[a-z0-9_]{3,32}$` | логин попадает в URL профиля |
| password_hash | text | NN, CHECK 20–255 | только хэш; нижняя граница отсекает случайно записанный открытый пароль |
| first_name | text | CHECK 1–64 | имя необязательно |
| gender | text | CHECK IN ('male','female') | |
| birth_date | date | CHECK > 1900-01-01 | возраст не хранится: он производный |
| bio | text | CHECK ≤ 2000 | «о себе» |
| is_private | boolean | NN, DEFAULT false | приватность профиля |
| avatar_file_id | bigint | FK → file, SET NULL | аватар — обычный файл из таблицы `file`, поэтому протокол и размер описаны там |
| deleted_at | timestamptz | | мягкое удаление: NULL — запись активна |
| — | — | CHECK deleted_at IS NULL OR (first_name, gender, birth_date, bio все NULL) | у удалённой записи персональных данных не остаётся: база не даст «удалить наполовину» |
| — | — | CHECK deleted_at IS NULL OR avatar_file_id IS NULL | аватар удалённого тоже стирается |
| created_at | timestamptz | NN, DEFAULT now() | момент регистрации |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер `account_set_updated_at` |

Функция `anonymize_account(id)` переписывает email и username на заглушку
`deleted_<id>`, обнуляет профиль и ставит `deleted_at`. Рецензии, оценки и
история остаются: внешние ключи на `account` из них — RESTRICT.

## 5. file — файл продукта

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| storage_key | text | UQ, CHECK 1–512 | бакет и путь к объекту в S3/MinIO, например `poster/shawshank-1.jpg` |
| external_url | text | UQ, CHECK `^[a-z][a-z0-9+.-]*://`, ≤ 1024 | адрес во внешнем сервисе. Проверяется только наличие схемы, потому что это может быть http, grpc или внутренний протокол хранилища |
| — | — | CHECK `num_nonnulls(storage_key, external_url) = 1` | файл либо у нас, либо снаружи, но не то и другое |
| mime_type | text | NN, CHECK `^[a-z]+/[a-z0-9.+-]{2,64}$` | image/jpeg, video/mp4 |
| size_bytes | bigint | CHECK > 0 | |
| — | — | CHECK storage_key IS NULL OR size_bytes IS NOT NULL | размер известен только для своих файлов |
| created_by | bigint | FK → account, SET NULL | служебное поле: кто загрузил файл |
| created_at | timestamptz | NN, DEFAULT now() | момент загрузки |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 6. film — карточка фильма или сериала

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| title | text | NN, CHECK 1–255 | |
| original_title | text | CHECK 1–255 | |
| film_type | text | NN, CHECK IN ('movie','series') | |
| — | — | **UQ (id, film_type)** | нужен, чтобы `season` ссылался на пару «фильм + тип» составным внешним ключом |
| production_year | smallint | NN, CHECK 1888–2200 | год производства, а не премьеры: снять могли в 2025, выпустить в 2027, а могли и не выпустить. Даты премьер живут в `film_release`, дублирования нет |
| duration_min | smallint | CHECK 1–6000 | только для фильма |
| — | — | CHECK film_type <> 'series' OR duration_min IS NULL | у сериала единой длительности нет: она в `episode`, среднее считается запросом |
| age_limit | smallint | NN, CHECK IN (0,6,12,16,18) | российская возрастная маркировка |
| description | text | CHECK ≤ 10000 | |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

Полей `poster_url` и `trailer_url` нет: постеры, кадры и трейлеры вынесены в
`film_file`, потому что их не может быть по одному.

## 7. film_file — файлы фильма

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| film_id | bigint | PK, FK → film, CASCADE | |
| file_id | bigint | PK, FK → file, RESTRICT | файл нельзя удалить, пока он используется фильмом |
| file_role | text | NN, CHECK IN ('poster','backdrop','still','trailer') | одна таблица на все виды файлов фильма |
| sort_order | smallint | NN, DEFAULT 1, CHECK > 0 | порядок показа внутри роли |
| — | — | UQ (film_id, file_role, sort_order) | второго «постера №1» быть не может, но постеров может быть много |
| created_at | timestamptz | NN, DEFAULT now() | `updated_at` нет: строка не изменяется |

## 8. genre — справочник жанров

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| name | text | NN, UQ, CHECK 2–64 | |
| slug | text | NN, UQ, CHECK `^[a-z0-9-]{2,64}$` | часть URL вида `/genre/drama` |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 9. film_genre — связь фильма и жанра

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| film_id | bigint | PK, FK → film, CASCADE | |
| genre_id | bigint | PK, FK → genre, RESTRICT | жанр нельзя удалить, пока он есть у фильмов |
| created_at | timestamptz | NN, DEFAULT now() | `updated_at` нет: связь либо есть, либо нет |

## 10. country — справочник стран

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| name | text | NN, UQ, CHECK 2–128 | |
| iso_code | text | NN, UQ, CHECK `^[A-Z]{2}$` | ISO 3166-1 alpha-2; маска вместо `char(2)`, который дополняет значения пробелами |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 11. film_country — страны производства

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| film_id | bigint | PK, FK → film, CASCADE | |
| country_id | bigint | PK, FK → country, RESTRICT | |
| created_at | timestamptz | NN, DEFAULT now() | `updated_at` нет |

## 12. similar_film — похожие фильмы

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| film_id | bigint | PK, FK → film, CASCADE | |
| similar_film_id | bigint | PK, FK → film, CASCADE | |
| — | — | **CHECK film_id < similar_film_id** | похожесть симметрична, поэтому пара хранится один раз в возрастающем порядке: (5, 9) и (9, 5) одновременно невозможны, ссылка на себя исключена тем же условием. На странице фильма обе стороны собираются одним UNION |
| created_at | timestamptz | NN, DEFAULT now() | |

## 13. film_release — премьеры

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| film_id | bigint | PK, FK → film, CASCADE | |
| country_id | bigint | PK, FK → country, RESTRICT | |
| release_type | text | PK, CHECK IN ('world_premiere','cinema','digital','tv') | в одной стране у фильма несколько дат: прокат, цифра, ТВ |
| release_date | date | **PK**, NN, CHECK 1888–2200 | дата входит в ключ, поэтому возможен повторный прокат: к юбилею или в отреставрированной версии |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

**Про 2НФ.** Отношение всеключевое: все четыре атрибута входят в первичный
ключ, неключевых атрибутов, кроме служебных меток времени, не остаётся, и
частичной зависимости быть не может по определению. Метки времени зависят от
всего ключа целиком: это время появления и правки именно этой строки о премьере.
Дата премьеры не может лежать полем в `film`, потому что она определяется тройкой
«фильм + страна + тип проката», а не одним фильмом.

## 14. season — сезон сериала

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| film_id | bigint | PK | |
| film_type | text | NN, DEFAULT 'series', CHECK = 'series' | поле существует только ради составного внешнего ключа |
| — | — | **FK (film_id, film_type) → film (id, film_type)**, CASCADE | правило «сезоны только у сериала» держит сама СУБД: у фильма сезон не создать, и сериал нельзя переключить в фильм, пока у него есть сезоны |
| season_number | smallint | PK, CHECK 0–100 | номер 0 — спецвыпуски |
| title | text | CHECK 1–255 | |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 15. episode — серия

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| film_id | bigint | PK, FK (составной) → season | |
| season_number | smallint | PK, FK (составной) → season, CASCADE | пара ссылается на сезон одним внешним ключом |
| episode_number | smallint | PK, CHECK 1–1000 | |
| title | text | NN, CHECK 1–255 | у серии название есть всегда |
| duration_min | smallint | CHECK 1–600 | настоящая длительность серии; средняя по сериалу считается запросом |
| release_date | date | CHECK 1888–2200 | |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 16. person — персона

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| first_name | text | NN, CHECK 1–128 | |
| last_name | text | NN, CHECK 1–128 | |
| birth_date | date | CHECK > 1800-01-01 | возраст вычисляется |
| photo_file_id | bigint | FK → file, SET NULL | фото — обычный файл |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 17. role_type — справочник кинопрофессий

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| code | text | NN, UQ, CHECK `^[a-z_]{3,32}$` | actor, director, writer, producer, composer, operator |
| title | text | NN, UQ, CHECK 2–64 | актёр, режиссёр, сценарист |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

Таблица называется `role_type`, а не `role`, потому что `role` занята ролью
пользователя в системе прав.

## 18. film_person — участие персоны в фильме

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| film_id | bigint | NN, FK → film, CASCADE | |
| person_id | bigint | NN, FK → person, CASCADE | |
| role_type_id | bigint | NN, FK → role_type, RESTRICT | |
| character_name | text | CHECK 1–255 | имя персонажа; заполняется только у актёров |
| — | — | UQ **NULLS NOT DISTINCT** (film_id, person_id, role_type_id, character_name) | обычный UNIQUE считает два NULL разными, и режиссёра можно было бы добавить дважды. Требует PostgreSQL 15+ |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

**Случай Стэна Ли.** Одна персона может участвовать в фильме в нескольких
профессиях сразу: строка со сценаристом (`role_type_id` = writer,
`character_name` пустой) и строка с актёром (`role_type_id` = actor,
`character_name` = «Стэн Ли») различаются значением `role_type_id`, поэтому
уникальный ключ их пропускает. Совпадение имени персонажа с именем самой
персоны схеме безразлично: `character_name` — обычный текст, никакой связи с
`person.first_name` у него нет. Если один актёр играет в фильме двух разных
персонажей, это тоже две строки, различающиеся `character_name`.

## 19. film_rating — оценка фильма

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| account_id | bigint | PK, FK → account, **RESTRICT** | оценки не исчезают вместе с учётной записью |
| film_id | bigint | PK, FK → film, CASCADE | |
| score | smallint | NN, CHECK 1–10 | шкала Кинопоиска |
| created_at | timestamptz | NN, DEFAULT now() | когда оценка поставлена впервые |
| updated_at | timestamptz | NN, DEFAULT now() | когда её последний раз меняли |

## 20. film_rating_history — история оценок
Требование ДЗ об историчности. Заполняется триггером, строки не изменяются.

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| account_id | bigint | NN, FK → account, **RESTRICT** | раньше каскад обнулял историю при удалении аккаунта |
| film_id | bigint | NN, FK → film, CASCADE | |
| old_score | smallint | CHECK 1–10 | пусто, если это первая оценка по фильму |
| new_score | smallint | NN, CHECK 1–10 | |
| — | — | CHECK old_score IS NULL OR old_score <> new_score | записей «было 7, стало 7» не бывает |
| created_at | timestamptz | NN, DEFAULT now() | момент изменения; `updated_at` нет, строки журнала не правятся |

## 21. review — рецензия

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| account_id | bigint | NN, FK → account, RESTRICT | |
| film_id | bigint | NN, FK → film, CASCADE | |
| title | text | NN, CHECK 3–255 | |
| content | text | NN, CHECK 50–20000 | нижняя граница отсекает «норм» в качестве рецензии |
| contains_spoilers | boolean | NN, DEFAULT false | текст скрывается до нажатия «показать» |
| deleted_at | timestamptz | | мягкое удаление |
| deleted_by | bigint | FK → account, RESTRICT | совпадает с account_id — удалил автор, иначе модератор |
| — | — | CHECK (deleted_at IS NULL) = (deleted_by IS NULL) | оба поля только вместе |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

Поля `is_approved` нет: состояние проверки живёт в `moderation_task`, чтобы не
хранить одно и то же в двух местах.

## 22. folder — папка пользователя

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| account_id | bigint | NN, FK → account, CASCADE | личные данные, уходят вместе с аккаунтом |
| title | text | NN, CHECK 1–64 | |
| folder_type | text | NN, CHECK IN ('favorite','watch_later','watched','person','custom') | |
| is_private | boolean | NN, DEFAULT true | |
| — | — | UQ (account_id, title) | |
| — | — | частичный UNIQUE (account_id, folder_type) WHERE folder_type <> 'custom' | системная папка каждого типа ровно одна, своих сколько угодно |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 23. folder_item — элемент папки

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| folder_id | bigint | NN, FK → folder, CASCADE | |
| film_id | bigint | FK → film, CASCADE | |
| person_id | bigint | FK → person, CASCADE | |
| — | — | CHECK `num_nonnulls(film_id, person_id) = 1` | в папке либо фильм, либо персона; обе ссылки настоящие внешние ключи |
| — | — | UQ (folder_id, film_id), UQ (folder_id, person_id) | дважды одно и то же в папку не добавить |
| created_at | timestamptz | NN, DEFAULT now() | по нему сортируется «недавно добавленное» |

## 24. release_subscription — «Напомнить о выходе»

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| account_id | bigint | PK, FK → account, CASCADE | |
| film_id | bigint | PK, FK → film, CASCADE | |
| created_at | timestamptz | NN, DEFAULT now() | |

## 25. collection — подборка

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| title | text | NN, CHECK 3–255 | |
| slug | text | NN, UQ, CHECK `^[a-z0-9-]{3,128}$` | адрес подборки |
| description | text | CHECK ≤ 2000 | |
| cover_file_id | bigint | FK → file, SET NULL | обложка — обычный файл |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 26. collection_film — фильмы в подборке

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| collection_id | bigint | PK, FK → collection, CASCADE | |
| film_id | bigint | PK, FK → film, CASCADE | |
| position | smallint | NN, CHECK > 0 | порядок показа |
| — | — | UQ (collection_id, position) DEFERRABLE | двух фильмов на одном месте нет, DEFERRABLE позволяет переставлять их внутри транзакции |
| created_at | timestamptz | NN, DEFAULT now() | |

## 27. notification — уведомление

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| account_id | bigint | NN, FK → account, CASCADE | личное, уходит вместе с аккаунтом |
| message | text | NN, CHECK 1–1000 | |
| is_read | boolean | NN, DEFAULT false | |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | меняется при отметке «прочитано» |

## 28. film_discussion — комната обсуждения

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| film_id | bigint | NN, UQ, FK → film, CASCADE | обсуждение одно на фильм |
| is_closed | boolean | NN, DEFAULT false | модератор может закрыть обсуждение |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

## 29. discussion_message — сообщение чата

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | по нему идёт постраничная выдача истории чата |
| discussion_id | bigint | NN, FK → film_discussion, CASCADE | |
| account_id | bigint | NN, FK → account, RESTRICT | |
| message_text | text | NN, CHECK 1–4000 | |
| deleted_at | timestamptz | | |
| deleted_by | bigint | FK → account, RESTRICT | различает «удалил автор» и «удалил модератор» |
| — | — | CHECK (deleted_at IS NULL) = (deleted_by IS NULL) | |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | сообщение можно отредактировать |

## 30. moderation_task — проверка контента

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| review_id | bigint | FK → review, CASCADE | |
| message_id | bigint | FK → discussion_message, CASCADE | |
| — | — | CHECK `num_nonnulls(review_id, message_id) = 1` | задача всегда про один объект |
| status | text | NN, DEFAULT 'pending', CHECK IN ('pending','approved','rejected') | публикуется то, что получило approved |
| source | text | NN, DEFAULT 'auto', CHECK IN ('auto','manual') | проверила модель или человек |
| model_name | text | CHECK 2–64 | какая модель и какой версии приняла решение |
| model_score | numeric(4,3) | CHECK 0–1 | уверенность модели; по порогу задача уходит на ручную проверку |
| decided_by | bigint | FK → account, RESTRICT | модератор, принявший решение |
| reason | text | CHECK 3–1000 | |
| decided_at | timestamptz | | |
| — | — | CHECK (status = 'pending') = (decided_at IS NULL) | нерешённая задача без даты решения, решённая — с датой |
| — | — | CHECK source <> 'auto' OR (model_name IS NOT NULL AND decided_by IS NULL) | автоматическая проверка всегда называет модель и не подписана человеком |
| — | — | CHECK source <> 'manual' OR status = 'pending' OR decided_by IS NOT NULL | решение человека всегда подписано |
| — | — | два частичных UNIQUE WHERE status = 'pending' | у объекта не может быть двух нерешённых задач |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

Модерация построена на одной таблице для рецензий и сообщений, поэтому
подключение нейросети не требует менять схему: модель пишет строку с
`source = 'auto'`, своим именем и оценкой уверенности, а спорные случаи
достаются человеку.

## 31. account_ban — блокировка пользователя
Крайняя мера модерации: пользователь лишается доступа на срок.

| Поле | Тип | Ограничения | Обоснование |
|---|---|---|---|
| id | bigint IDENTITY | PK | |
| account_id | bigint | NN, FK → account, RESTRICT | кого заблокировали |
| moderator_id | bigint | NN, FK → account, RESTRICT | кто заблокировал; решение должно оставаться подписанным |
| reason | text | NN, CHECK 3–1000 | причина обязательна |
| active_during | `tstzrange` | NN, CHECK NOT isempty | период действия; пустая верхняя граница означает бессрочную блокировку |
| — | — | CHECK account_id <> moderator_id | модератор не блокирует сам себя |
| — | — | **EXCLUDE USING gist (account_id WITH =, active_during WITH &&)** | у одного пользователя блокировки не пересекаются во времени, иначе нельзя ответить, какая действует сейчас. Обычный UNIQUE сравнивать диапазоны не умеет, поэтому нужен EXCLUDE и расширение `btree_gist` |
| created_at | timestamptz | NN, DEFAULT now() | |
| updated_at | timestamptz | NN, DEFAULT now() | ведёт триггер |

Диапазон `tstzrange` — скалярное значение встроенного типа, а не составное:
его нельзя разложить на переменное число элементов, он сравнивается и
индексируется целиком. Пара полей «начало» и «конец» здесь не годится —
на неё нельзя наложить EXCLUDE.

---

## 32. Хранилища вне PostgreSQL

### Redis

| Ключ | Структура | Значение | TTL | Почему не в PostgreSQL |
|---|---|---|---|---|
| `session:<session_id>` | строка | account_id | 30 дней | читается на каждом запросе; потеря не критична, пользователь перелогинится |
| `email_verification:<token>` | строка | account_id | 24 часа | одноразовый токен, удаляется сам |
| `password_reset:<token>` | строка | account_id | 15 минут | короткий срок жизни — требование безопасности |
| `ratelimit:login:<ip>` | счётчик | число попыток | 15 минут | защита от перебора пароля |
| `cache:film_top` | строка | сериализованный топ | 1 час | кэш тяжёлого запроса со средними оценками |

### S3 / MinIO
Все файлы описаны таблицей `file`: в ней либо `storage_key` (бакет и путь к
объекту), либо `external_url` (внешний сервис). Бинарные данные в базу не
попадают.

| Бакет | Что хранит |
|---|---|
| `avatar` | аватары пользователей |
| `poster` | постеры и широкие обложки фильмов |
| `still` | кадры из фильмов |
| `person` | фотографии персон |
| `cover` | обложки подборок |

## 33. Сводка по типам и их отображение в Go

Таблица отвечает на вопрос ревью о том, как типы SQL превращаются в типы Go при
работе через `pgx`.

| Тип PostgreSQL | Тип Go | Если поле может быть NULL |
|---|---|---|
| `bigint` | `int64` | `pgtype.Int8` |
| `smallint` | `int16` | `pgtype.Int2` |
| `text` | `string` | `pgtype.Text` |
| `boolean` | `bool` | `pgtype.Bool` |
| `timestamptz` | `time.Time` | `pgtype.Timestamptz` |
| `date` | `time.Time` | `pgtype.Date` |
| `numeric(4,3)` | `pgtype.Numeric` или `decimal.Decimal` | `pgtype.Numeric` |
| `tstzrange` | `pgtype.Range[time.Time]` | `pgtype.Range[time.Time]` |

Правило простое: обязательное поле читается в обычный тип Go, необязательное —
в тип из `pgtype`, иначе NULL превратится в ноль или пустую строку и отличить
его будет нельзя. `numeric` не читается в `float64`: двоичная плавающая точка
теряет точность, а `pgtype.Numeric` хранит значение как есть.

Не используются: `varchar(n)`, `char(n)`, `serial`, `money`, `timestamp` без
часового пояса, `array`, `json`, `jsonb`.

## 34. Сводка по ограничениям целостности

| Ограничение | Где |
|---|---|
| PRIMARY KEY | у всех 31 таблицы |
| UNIQUE | email, username, коды и названия справочников, slug, (id, film_type) у film, (film_id, file_role, sort_order) у film_file, film_id у обсуждения |
| UNIQUE NULLS NOT DISTINCT | film_person |
| Частичные уникальные индексы | системные папки, нерешённые задачи модерации по рецензии и по сообщению |
| FOREIGN KEY | все ссылки; RESTRICT на пользовательский контент, CASCADE на личные данные, SET NULL на файлы; `ON UPDATE` не указан нигде |
| Составной FOREIGN KEY | season → film (id, film_type); episode → season (film_id, season_number) |
| NOT NULL | все обязательные поля |
| CHECK | маски, диапазоны, длины, `num_nonnulls`, симметричность similar_film, согласованность пар полей, анонимизация аккаунта |
| DEFAULT | флаги, статусы, created_at, updated_at |
| Триггеры | set_updated_at на всех изменяемых таблицах, write_film_rating_history |
| Функции | anonymize_account(bigint) — мягкое удаление учётной записи |

| EXCLUDE | account_ban: блокировки одного пользователя не пересекаются во времени |

Расширение `btree_gist` подключается первой миграцией -- без него в одном
ограничении EXCLUDE нельзя совместить сравнение идентификатора на равенство
и диапазона на пересечение.
