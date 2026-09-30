# ER-диаграммы

Основные диаграммы — в **нотации Чена**: сущность рисуется прямоугольником,
связь — ромбом, атрибут — овалом, ключевой атрибут подчёркнут. Двойной контур
означает слабую сущность и идентифицирующую связь: такая сущность не существует
отдельно от родителя и опознаётся только вместе с ним.

Кардинальность подписана на линиях: 1 — одна сторона, M и N — многие, P — третья
сторона тернарной связи.

Файлы лежат в [chen/](chen), в форматах PNG и SVG, исходники — в `.dot`.

---

## 1. Общая карта

Все 31 отношение схемы. Связующие таблицы (`film_genre`, `film_person`,
`collection_film` и другие) в нотации Чена показаны именно как связи-ромбы, а не
как сущности: в реляционной схеме они превращаются в таблицы, но на ER-уровне
это связи.

![Общая карта](chen/chen-overview.png)

## 2. Пользователи, права доступа, файлы

![Пользователи и права](chen/chen-account.png)

Роль пользователя и набор прав вынесены в отдельные сущности: права настраиваются
данными, а не зашиты в код. Блокировка `account_ban` хранит период действия
диапазоном `tstzrange`, на который наложено ограничение EXCLUDE: пересекающихся
блокировок одного пользователя быть не может. Файл — самостоятельная сущность со своими
служебными атрибутами, поэтому у аватара, постера и трейлера общее хранилище.

## 3. Каталог, сезоны и серии

![Каталог](chen/chen-catalog.png)

`season` и `episode` — слабые сущности: номер сезона имеет смысл только внутри
сериала, а номер серии — только внутри сезона. Связь «премьера» несёт
собственные атрибуты `release_type` и `release_date`, связь «файлы_фильма» —
атрибуты `file_role` и `sort_order`, за счёт чего у фильма может быть несколько
постеров и трейлеров.

## 4. Фильм, персона, профессия

![Персоны](chen/chen-person.png)

Связь «участие» тернарная: она соединяет фильм, персону и профессию и несёт
атрибут `character_name`. Поэтому один человек может быть в фильме одновременно
сценаристом и актёром, а один актёр — сыграть несколько персонажей.

## 5. Оценки, рецензии, папки, чат и модерация

![Активность](chen/chen-activity.png)

`moderation_task` — самостоятельная сущность: проверку проходит и рецензия, и
сообщение чата, решение принимает либо модель, либо модератор.

---

## Приложение: те же связи в виде «вороньей лапки»

Диаграммы ниже показывают ту же схему в виде таблиц со столбцами: так удобнее
сверяться с DDL. Синтаксис Mermaid, рендерится на GitHub. Типы данных не
указаны — они в [fields.md](fields.md).

### Пользователи, права, файлы

```mermaid
erDiagram
    role ||--o{ account : "назначена"
    account ||--o{ account_ban : "заблокирован"
    account ||--o{ account_ban : "выдал блокировку"
    role ||--o{ role_permission : ""
    permission ||--o{ role_permission : ""
    account ||--o{ file : "загрузил"
    file ||--o{ account : "аватар"

    role {
        attr id PK
        attr code UK
        attr title UK
    }
    permission {
        attr id PK
        attr code UK
        attr title UK
    }
    role_permission {
        attr role_id PK
        attr permission_id PK
    }
    account {
        attr id PK
        attr role_id FK
        attr email UK
        attr username UK
        attr password_hash
        attr first_name
        attr gender
        attr birth_date
        attr bio
        attr is_private
        attr avatar_file_id FK
        attr deleted_at
    }
    file {
        attr id PK
        attr storage_key UK
        attr external_url UK
        attr mime_type
        attr size_bytes
        attr created_by FK
    }
    account_ban {
        attr id PK
        attr account_id FK
        attr moderator_id FK
        attr reason
        attr active_during
    }
```

### Каталог

```mermaid
erDiagram
    film ||--o{ film_file : ""
    file ||--o{ film_file : ""
    film ||--o{ film_genre : ""
    genre ||--o{ film_genre : ""
    film ||--o{ film_country : ""
    country ||--o{ film_country : ""
    film ||--o{ film_release : ""
    country ||--o{ film_release : ""
    film ||--o{ similar_film : ""
    film ||--o{ season : ""
    season ||--o{ episode : ""
    film ||--o{ film_person : ""
    person ||--o{ film_person : ""
    role_type ||--o{ film_person : ""

    film {
        attr id PK
        attr title
        attr original_title
        attr film_type
        attr production_year
        attr duration_min
        attr age_limit
        attr description
    }
    film_file {
        attr film_id PK
        attr file_id PK
        attr file_role
        attr sort_order
    }
    film_genre {
        attr film_id PK
        attr genre_id PK
    }
    film_country {
        attr film_id PK
        attr country_id PK
    }
    similar_film {
        attr film_id PK
        attr similar_film_id PK
    }
    film_release {
        attr film_id PK
        attr country_id PK
        attr release_type PK
        attr release_date PK
    }
    season {
        attr film_id PK
        attr film_type FK
        attr season_number PK
        attr title
    }
    episode {
        attr film_id PK
        attr season_number PK
        attr episode_number PK
        attr title
        attr duration_min
        attr release_date
    }
    person {
        attr id PK
        attr first_name
        attr last_name
        attr birth_date
        attr photo_file_id FK
    }
    role_type {
        attr id PK
        attr code UK
        attr title UK
    }
    film_person {
        attr id PK
        attr film_id FK
        attr person_id FK
        attr role_type_id FK
        attr character_name
    }
```

### Активность, чат, модерация

```mermaid
erDiagram
    account ||--o{ film_rating : ""
    film ||--o{ film_rating : ""
    account ||--o{ film_rating_history : ""
    film ||--o{ film_rating_history : ""
    account ||--o{ review : ""
    film ||--o{ review : ""
    account ||--o{ folder : ""
    folder ||--o{ folder_item : ""
    film ||--o{ folder_item : ""
    person ||--o{ folder_item : ""
    account ||--o{ release_subscription : ""
    film ||--o{ release_subscription : ""
    collection ||--o{ collection_film : ""
    film ||--o{ collection_film : ""
    account ||--o{ notification : ""
    film ||--|| film_discussion : ""
    film_discussion ||--o{ discussion_message : ""
    account ||--o{ discussion_message : ""
    review ||--o{ moderation_task : ""
    discussion_message ||--o{ moderation_task : ""
    account ||--o{ moderation_task : "решил"

    film_rating {
        attr account_id PK
        attr film_id PK
        attr score
    }
    film_rating_history {
        attr id PK
        attr account_id FK
        attr film_id FK
        attr old_score
        attr new_score
    }
    review {
        attr id PK
        attr account_id FK
        attr film_id FK
        attr title
        attr content
        attr contains_spoilers
        attr deleted_at
        attr deleted_by FK
    }
    folder {
        attr id PK
        attr account_id FK
        attr title
        attr folder_type
        attr is_private
    }
    folder_item {
        attr id PK
        attr folder_id FK
        attr film_id FK
        attr person_id FK
    }
    release_subscription {
        attr account_id PK
        attr film_id PK
    }
    collection {
        attr id PK
        attr title
        attr slug UK
        attr description
        attr cover_file_id FK
    }
    collection_film {
        attr collection_id PK
        attr film_id PK
        attr position
    }
    notification {
        attr id PK
        attr account_id FK
        attr message
        attr is_read
    }
    film_discussion {
        attr id PK
        attr film_id UK
        attr is_closed
    }
    discussion_message {
        attr id PK
        attr discussion_id FK
        attr account_id FK
        attr message_text
        attr deleted_at
        attr deleted_by FK
    }
    moderation_task {
        attr id PK
        attr review_id FK
        attr message_id FK
        attr status
        attr source
        attr model_name
        attr model_score
        attr decided_by FK
        attr reason
        attr decided_at
    }
```

### Хранилища вне PostgreSQL

Внешних ключей между разными СУБД нет, связь логическая: целостность
поддерживает приложение.

```mermaid
erDiagram
    account ||..o{ session : "Redis"
    account ||..o| email_verification : "Redis"
    account ||..o| password_reset : "Redis"

    session {
        attr session_id PK "TTL 30 дней"
        attr account_id
    }
    email_verification {
        attr token PK "TTL 24 часа"
        attr account_id
    }
    password_reset {
        attr token PK "TTL 15 минут"
        attr account_id
    }
    rate_limit {
        attr ip_address PK "TTL 15 минут"
        attr attempt_count
    }
```

Кэш топа фильмов (`cache:film_top`, TTL 1 час) в виде таблицы не представим: это
сериализованный список, который перезаписывается целиком.

В S3/MinIO лежат сами файлы, в PostgreSQL — только отношение `file` со ссылками.
Бакеты: `avatar`, `poster`, `still`, `person`, `cover`.
