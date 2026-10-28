CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE FUNCTION set_updated_at() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

CREATE TABLE role (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       text        NOT NULL,
    title      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT role_code_key UNIQUE (code),
    CONSTRAINT role_title_key UNIQUE (title),
    CONSTRAINT role_code_check CHECK (code ~ '^[a-z_]{3,32}$'),
    CONSTRAINT role_title_check CHECK (length(title) BETWEEN 2 AND 64)
);

CREATE TRIGGER role_set_updated_at
    BEFORE UPDATE ON role
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE role IS 'Роль пользователя: читатель, модератор, редактор, администратор.';

CREATE TABLE permission (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       text        NOT NULL,
    title      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT permission_code_key UNIQUE (code),
    CONSTRAINT permission_title_key UNIQUE (title),
    CONSTRAINT permission_code_check CHECK (code ~ '^[a-z_.]{3,64}$'),
    CONSTRAINT permission_title_check CHECK (length(title) BETWEEN 2 AND 128)
);

CREATE TRIGGER permission_set_updated_at
    BEFORE UPDATE ON permission
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE permission IS
    'Отдельное право: написать рецензию, модерировать контент, править каталог, загрузить файл.';

CREATE TABLE role_permission (
    role_id       bigint      NOT NULL,
    permission_id bigint      NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT role_permission_pkey PRIMARY KEY (role_id, permission_id),
    CONSTRAINT role_permission_role_id_fkey FOREIGN KEY (role_id)
        REFERENCES role (id) ON DELETE CASCADE,
    CONSTRAINT role_permission_permission_id_fkey FOREIGN KEY (permission_id)
        REFERENCES permission (id) ON DELETE CASCADE
);

COMMENT ON TABLE role_permission IS
    'Какие права даёт роль. Набор прав настраивается данными, а не кодом приложения.';

CREATE TABLE account (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    role_id       bigint      NOT NULL,
    email         text        NOT NULL,
    username      text        NOT NULL,
    password_hash text        NOT NULL,
    first_name    text,
    gender        text,
    birth_date    date,
    bio           text,
    is_private    boolean     NOT NULL DEFAULT false,
    deleted_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT account_email_key UNIQUE (email),
    CONSTRAINT account_username_key UNIQUE (username),
    CONSTRAINT account_role_id_fkey FOREIGN KEY (role_id)
        REFERENCES role (id) ON DELETE RESTRICT,
    CONSTRAINT account_email_check
        CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'
               AND length(email) <= 254),
    CONSTRAINT account_username_check CHECK (username ~ '^[a-z0-9_]{3,32}$'),
    CONSTRAINT account_password_hash_check
        CHECK (length(password_hash) BETWEEN 20 AND 255),
    CONSTRAINT account_first_name_check CHECK (length(first_name) BETWEEN 1 AND 64),
    CONSTRAINT account_gender_check CHECK (gender IN ('male', 'female')),
    CONSTRAINT account_birth_date_check CHECK (birth_date > DATE '1900-01-01'),
    CONSTRAINT account_bio_check CHECK (length(bio) <= 2000),
    CONSTRAINT account_anonymized_check
        CHECK (deleted_at IS NULL
               OR (first_name IS NULL AND gender IS NULL AND birth_date IS NULL
                   AND bio IS NULL))
);

CREATE TRIGGER account_set_updated_at
    BEFORE UPDATE ON account
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE account IS
    'Учётная запись и профиль. Удаление мягкое: заполняется deleted_at, персональные данные стираются.';
COMMENT ON COLUMN account.deleted_at IS
    'Момент удаления учётной записи. NULL — запись активна.';

CREATE TABLE file (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    storage_key  text,
    external_url text,
    mime_type    text        NOT NULL,
    size_bytes   bigint,
    created_by   bigint,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT file_storage_key_key UNIQUE (storage_key),
    CONSTRAINT file_external_url_key UNIQUE (external_url),
    CONSTRAINT file_created_by_fkey FOREIGN KEY (created_by)
        REFERENCES account (id) ON DELETE SET NULL,
    CONSTRAINT file_location_check CHECK (num_nonnulls(storage_key, external_url) = 1),
    CONSTRAINT file_storage_key_check CHECK (length(storage_key) BETWEEN 1 AND 512),
    CONSTRAINT file_external_url_check
        CHECK (external_url ~ '^[a-z][a-z0-9+.-]*://' AND length(external_url) <= 1024),
    CONSTRAINT file_mime_type_check CHECK (mime_type ~ '^[a-z]+/[a-z0-9.+-]{2,64}$'),
    CONSTRAINT file_size_bytes_check CHECK (size_bytes > 0),
    CONSTRAINT file_size_known_check
        CHECK (storage_key IS NULL OR size_bytes IS NOT NULL)
);

CREATE TRIGGER file_set_updated_at
    BEFORE UPDATE ON file
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE file IS
    'Файл продукта: ключ объекта в S3/MinIO либо ссылка на внешний сервис, плюс служебные поля.';
COMMENT ON COLUMN file.created_by IS
    'Кто загрузил файл. NULL, если учётная запись загрузившего удалена.';

ALTER TABLE account
    ADD COLUMN avatar_file_id bigint,
    ADD CONSTRAINT account_avatar_file_id_fkey FOREIGN KEY (avatar_file_id)
        REFERENCES file (id) ON DELETE SET NULL,
    ADD CONSTRAINT account_avatar_anonymized_check
        CHECK (deleted_at IS NULL OR avatar_file_id IS NULL);

CREATE FUNCTION anonymize_account(target_id bigint) RETURNS void
    LANGUAGE plpgsql AS $$
BEGIN
    UPDATE account
       SET email          = 'deleted_' || target_id || '@deleted.invalid',
           username       = 'deleted_' || target_id,
           password_hash  = repeat('x', 20),
           first_name     = NULL,
           gender         = NULL,
           birth_date     = NULL,
           bio            = NULL,
           avatar_file_id = NULL,
           is_private     = true,
           deleted_at     = now()
     WHERE id = target_id
       AND deleted_at IS NULL;
END;
$$;

COMMENT ON FUNCTION anonymize_account(bigint) IS
    'Мягкое удаление учётной записи: персональные данные стираются, рецензии и оценки остаются.';

CREATE TABLE film (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title           text        NOT NULL,
    original_title  text,
    film_type       text        NOT NULL,
    production_year smallint    NOT NULL,
    duration_min    smallint,
    age_limit       smallint    NOT NULL,
    description     text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_id_film_type_key UNIQUE (id, film_type),
    CONSTRAINT film_film_type_check CHECK (film_type IN ('movie', 'series')),
    CONSTRAINT film_title_check CHECK (length(title) BETWEEN 1 AND 255),
    CONSTRAINT film_original_title_check CHECK (length(original_title) BETWEEN 1 AND 255),
    CONSTRAINT film_production_year_check CHECK (production_year BETWEEN 1888 AND 2200),
    CONSTRAINT film_duration_min_check CHECK (duration_min BETWEEN 1 AND 6000),
    CONSTRAINT film_duration_series_check
        CHECK (film_type <> 'series' OR duration_min IS NULL),
    CONSTRAINT film_age_limit_check CHECK (age_limit IN (0, 6, 12, 16, 18)),
    CONSTRAINT film_description_check CHECK (length(description) <= 10000)
);

CREATE TRIGGER film_set_updated_at
    BEFORE UPDATE ON film
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE film IS
    'Карточка фильма или сериала. Тип задаётся полем film_type.';
COMMENT ON COLUMN film.production_year IS
    'Год производства, а не дата премьеры: премьеры лежат в film_release.';
COMMENT ON COLUMN film.duration_min IS
    'Только для фильма. У сериала длительность хранится в episode.';

CREATE TABLE film_file (
    film_id    bigint      NOT NULL,
    file_id    bigint      NOT NULL,
    file_role  text        NOT NULL,
    sort_order smallint    NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_file_pkey PRIMARY KEY (film_id, file_id),
    CONSTRAINT film_file_order_key UNIQUE (film_id, file_role, sort_order),
    CONSTRAINT film_file_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_file_file_id_fkey FOREIGN KEY (file_id)
        REFERENCES file (id) ON DELETE RESTRICT,
    CONSTRAINT film_file_file_role_check
        CHECK (file_role IN ('poster', 'backdrop', 'still', 'trailer')),
    CONSTRAINT film_file_sort_order_check CHECK (sort_order > 0)
);

COMMENT ON TABLE film_file IS
    'Файлы фильма: постеры, кадры, трейлеры. Роль и порядок показа задаются здесь.';

CREATE TABLE genre (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       text        NOT NULL,
    slug       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT genre_name_key UNIQUE (name),
    CONSTRAINT genre_slug_key UNIQUE (slug),
    CONSTRAINT genre_name_check CHECK (length(name) BETWEEN 2 AND 64),
    CONSTRAINT genre_slug_check CHECK (slug ~ '^[a-z0-9-]{2,64}$')
);

CREATE TRIGGER genre_set_updated_at
    BEFORE UPDATE ON genre
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE genre IS 'Справочник жанров.';

CREATE TABLE film_genre (
    film_id    bigint      NOT NULL,
    genre_id   bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_genre_pkey PRIMARY KEY (film_id, genre_id),
    CONSTRAINT film_genre_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_genre_genre_id_fkey FOREIGN KEY (genre_id)
        REFERENCES genre (id) ON DELETE RESTRICT
);

COMMENT ON TABLE film_genre IS 'Связь фильма и жанра, многие ко многим.';

CREATE TABLE country (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       text        NOT NULL,
    iso_code   text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT country_name_key UNIQUE (name),
    CONSTRAINT country_iso_code_key UNIQUE (iso_code),
    CONSTRAINT country_name_check CHECK (length(name) BETWEEN 2 AND 128),
    CONSTRAINT country_iso_code_check CHECK (iso_code ~ '^[A-Z]{2}$')
);

CREATE TRIGGER country_set_updated_at
    BEFORE UPDATE ON country
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE country IS 'Справочник стран, код по ISO 3166-1 alpha-2.';

CREATE TABLE film_country (
    film_id    bigint      NOT NULL,
    country_id bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_country_pkey PRIMARY KEY (film_id, country_id),
    CONSTRAINT film_country_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_country_country_id_fkey FOREIGN KEY (country_id)
        REFERENCES country (id) ON DELETE RESTRICT
);

COMMENT ON TABLE film_country IS 'Страны производства фильма, многие ко многим.';

CREATE TABLE similar_film (
    film_id         bigint      NOT NULL,
    similar_film_id bigint      NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT similar_film_pkey PRIMARY KEY (film_id, similar_film_id),
    CONSTRAINT similar_film_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT similar_film_similar_film_id_fkey FOREIGN KEY (similar_film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT similar_film_order_check CHECK (film_id < similar_film_id)
);

COMMENT ON TABLE similar_film IS
    'Похожие фильмы. Одна строка на пару, симметричность обеспечена CHECK film_id < similar_film_id.';

CREATE TABLE film_release (
    film_id      bigint      NOT NULL,
    country_id   bigint      NOT NULL,
    release_type text        NOT NULL,
    release_date date        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_release_pkey
        PRIMARY KEY (film_id, country_id, release_type, release_date),
    CONSTRAINT film_release_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_release_country_id_fkey FOREIGN KEY (country_id)
        REFERENCES country (id) ON DELETE RESTRICT,
    CONSTRAINT film_release_release_type_check
        CHECK (release_type IN ('world_premiere', 'cinema', 'digital', 'tv')),
    CONSTRAINT film_release_release_date_check
        CHECK (release_date >= DATE '1888-01-01' AND release_date <= DATE '2200-01-01')
);

CREATE TRIGGER film_release_set_updated_at
    BEFORE UPDATE ON film_release
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE film_release IS
    'Даты премьер по странам и типам проката, включая повторный прокат.';

CREATE TABLE season (
    film_id       bigint      NOT NULL,
    film_type     text        NOT NULL DEFAULT 'series',
    season_number smallint    NOT NULL,
    title         text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT season_pkey PRIMARY KEY (film_id, season_number),
    CONSTRAINT season_film_fkey FOREIGN KEY (film_id, film_type)
        REFERENCES film (id, film_type) ON DELETE CASCADE,
    CONSTRAINT season_film_type_check CHECK (film_type = 'series'),
    CONSTRAINT season_season_number_check CHECK (season_number BETWEEN 0 AND 100),
    CONSTRAINT season_title_check CHECK (length(title) BETWEEN 1 AND 255)
);

CREATE TRIGGER season_set_updated_at
    BEFORE UPDATE ON season
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE season IS
    'Сезон сериала. Правило "сезоны только у сериала" держит составной внешний ключ на (id, film_type).';
COMMENT ON COLUMN season.film_type IS
    'Всегда series. Поле существует только ради составного внешнего ключа на film.';

CREATE TABLE episode (
    film_id        bigint      NOT NULL,
    season_number  smallint    NOT NULL,
    episode_number smallint    NOT NULL,
    title          text        NOT NULL,
    duration_min   smallint,
    release_date   date,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT episode_pkey PRIMARY KEY (film_id, season_number, episode_number),
    CONSTRAINT episode_season_fkey FOREIGN KEY (film_id, season_number)
        REFERENCES season (film_id, season_number) ON DELETE CASCADE,
    CONSTRAINT episode_episode_number_check CHECK (episode_number BETWEEN 1 AND 1000),
    CONSTRAINT episode_title_check CHECK (length(title) BETWEEN 1 AND 255),
    CONSTRAINT episode_duration_min_check CHECK (duration_min BETWEEN 1 AND 600),
    CONSTRAINT episode_release_date_check
        CHECK (release_date >= DATE '1888-01-01' AND release_date <= DATE '2200-01-01')
);

CREATE TRIGGER episode_set_updated_at
    BEFORE UPDATE ON episode
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE episode IS 'Серия сезона. Здесь хранится настоящая длительность серии.';

CREATE TABLE person (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    first_name    text        NOT NULL,
    last_name     text        NOT NULL,
    birth_date    date,
    photo_file_id bigint,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT person_photo_file_id_fkey FOREIGN KEY (photo_file_id)
        REFERENCES file (id) ON DELETE SET NULL,
    CONSTRAINT person_first_name_check CHECK (length(first_name) BETWEEN 1 AND 128),
    CONSTRAINT person_last_name_check CHECK (length(last_name) BETWEEN 1 AND 128),
    CONSTRAINT person_birth_date_check CHECK (birth_date > DATE '1800-01-01')
);

CREATE TRIGGER person_set_updated_at
    BEFORE UPDATE ON person
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE person IS
    'Персона: актёр, режиссёр, сценарист. Роль в конкретном фильме хранится в film_person.';

CREATE TABLE role_type (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       text        NOT NULL,
    title      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT role_type_code_key UNIQUE (code),
    CONSTRAINT role_type_title_key UNIQUE (title),
    CONSTRAINT role_type_code_check CHECK (code ~ '^[a-z_]{3,32}$'),
    CONSTRAINT role_type_title_check CHECK (length(title) BETWEEN 2 AND 64)
);

CREATE TRIGGER role_type_set_updated_at
    BEFORE UPDATE ON role_type
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE role_type IS
    'Справочник кинопрофессий: актёр, режиссёр, сценарист, продюсер, композитор, оператор.';

CREATE TABLE film_person (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    film_id        bigint      NOT NULL,
    person_id      bigint      NOT NULL,
    role_type_id   bigint      NOT NULL,
    character_name text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_person_unique
        UNIQUE NULLS NOT DISTINCT (film_id, person_id, role_type_id, character_name),
    CONSTRAINT film_person_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_person_person_id_fkey FOREIGN KEY (person_id)
        REFERENCES person (id) ON DELETE CASCADE,
    CONSTRAINT film_person_role_type_id_fkey FOREIGN KEY (role_type_id)
        REFERENCES role_type (id) ON DELETE RESTRICT,
    CONSTRAINT film_person_character_name_check
        CHECK (length(character_name) BETWEEN 1 AND 255)
);

CREATE TRIGGER film_person_set_updated_at
    BEFORE UPDATE ON film_person
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE film_person IS
    'Участие персоны в фильме. character_name заполняется только у актёров.';

CREATE TABLE film_rating (
    account_id bigint      NOT NULL,
    film_id    bigint      NOT NULL,
    score      smallint    NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_rating_pkey PRIMARY KEY (account_id, film_id),
    CONSTRAINT film_rating_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT film_rating_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_rating_score_check CHECK (score BETWEEN 1 AND 10)
);

CREATE TRIGGER film_rating_set_updated_at
    BEFORE UPDATE ON film_rating
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE film_rating IS
    'Оценка фильма пользователем от 1 до 10, одна на пару пользователь-фильм.';

CREATE TABLE film_rating_history (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint      NOT NULL,
    film_id    bigint      NOT NULL,
    old_score  smallint,
    new_score  smallint    NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_rating_history_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT film_rating_history_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_rating_history_old_score_check CHECK (old_score BETWEEN 1 AND 10),
    CONSTRAINT film_rating_history_new_score_check CHECK (new_score BETWEEN 1 AND 10),
    CONSTRAINT film_rating_history_change_check
        CHECK (old_score IS NULL OR old_score <> new_score)
);

COMMENT ON TABLE film_rating_history IS
    'История изменения оценок: было 7, стало 9 (историчность данных).';
COMMENT ON COLUMN film_rating_history.old_score IS
    'Пусто, если это первая оценка пользователя по данному фильму.';

CREATE FUNCTION write_film_rating_history() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    IF tg_op = 'INSERT' THEN
        INSERT INTO film_rating_history (account_id, film_id, old_score, new_score)
        VALUES (NEW.account_id, NEW.film_id, NULL, NEW.score);
    ELSIF NEW.score <> OLD.score THEN
        INSERT INTO film_rating_history (account_id, film_id, old_score, new_score)
        VALUES (NEW.account_id, NEW.film_id, OLD.score, NEW.score);
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER film_rating_write_history
    AFTER INSERT OR UPDATE ON film_rating
    FOR EACH ROW EXECUTE FUNCTION write_film_rating_history();

CREATE TABLE review (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id        bigint      NOT NULL,
    film_id           bigint      NOT NULL,
    title             text        NOT NULL,
    content           text        NOT NULL,
    contains_spoilers boolean     NOT NULL DEFAULT false,
    deleted_at        timestamptz,
    deleted_by        bigint,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT review_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT review_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT review_deleted_by_fkey FOREIGN KEY (deleted_by)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT review_title_check CHECK (length(title) BETWEEN 3 AND 255),
    CONSTRAINT review_content_check CHECK (length(content) BETWEEN 50 AND 20000),
    CONSTRAINT review_deleted_pair_check
        CHECK ((deleted_at IS NULL) = (deleted_by IS NULL))
);

CREATE UNIQUE INDEX review_active_unique_idx
    ON review (account_id, film_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER review_set_updated_at
    BEFORE UPDATE ON review
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE review IS
    'Рецензия пользователя на фильм. Одна действующая рецензия на пару пользователь-фильм; после удаления можно написать новую.';
COMMENT ON INDEX review_active_unique_idx IS
    'Ограничение целостности: у пользователя не бывает двух неудалённых рецензий на один фильм. Удалённые рецензии остаются в таблице и повторной записи не мешают.';
COMMENT ON COLUMN review.deleted_by IS
    'Кто удалил рецензию: если совпадает с account_id — сам автор, иначе модератор.';

CREATE TABLE folder (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id  bigint      NOT NULL,
    title       text        NOT NULL,
    folder_type text        NOT NULL,
    is_private  boolean     NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT folder_account_id_title_key UNIQUE (account_id, title),
    CONSTRAINT folder_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE CASCADE,
    CONSTRAINT folder_title_check CHECK (length(title) BETWEEN 1 AND 64),
    CONSTRAINT folder_folder_type_check
        CHECK (folder_type IN ('favorite', 'watch_later', 'watched', 'person', 'custom'))
);

CREATE UNIQUE INDEX folder_system_unique_idx
    ON folder (account_id, folder_type)
    WHERE folder_type <> 'custom';

CREATE TRIGGER folder_set_updated_at
    BEFORE UPDATE ON folder
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE folder IS
    'Папка пользователя: Избранное, Буду смотреть, Просмотренные, папка с персонами, свои папки.';

CREATE TABLE folder_item (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    folder_id  bigint      NOT NULL,
    film_id    bigint,
    person_id  bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT folder_item_folder_id_fkey FOREIGN KEY (folder_id)
        REFERENCES folder (id) ON DELETE CASCADE,
    CONSTRAINT folder_item_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT folder_item_person_id_fkey FOREIGN KEY (person_id)
        REFERENCES person (id) ON DELETE CASCADE,
    CONSTRAINT folder_item_target_check CHECK (num_nonnulls(film_id, person_id) = 1),
    CONSTRAINT folder_item_film_key UNIQUE (folder_id, film_id),
    CONSTRAINT folder_item_person_key UNIQUE (folder_id, person_id)
);

COMMENT ON TABLE folder_item IS
    'Элемент папки: фильм или персона. Обе ссылки — настоящие внешние ключи, выбор между ними задаёт CHECK.';

CREATE TABLE release_subscription (
    account_id bigint      NOT NULL,
    film_id    bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT release_subscription_pkey PRIMARY KEY (account_id, film_id),
    CONSTRAINT release_subscription_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE CASCADE,
    CONSTRAINT release_subscription_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE
);

COMMENT ON TABLE release_subscription IS 'Подписка на выход фильма: "Напомнить о выходе".';

CREATE TABLE collection (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title          text        NOT NULL,
    slug           text        NOT NULL,
    description    text,
    cover_file_id  bigint,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT collection_slug_key UNIQUE (slug),
    CONSTRAINT collection_cover_file_id_fkey FOREIGN KEY (cover_file_id)
        REFERENCES file (id) ON DELETE SET NULL,
    CONSTRAINT collection_title_check CHECK (length(title) BETWEEN 3 AND 255),
    CONSTRAINT collection_slug_check CHECK (slug ~ '^[a-z0-9-]{3,128}$'),
    CONSTRAINT collection_description_check CHECK (length(description) <= 2000)
);

CREATE TRIGGER collection_set_updated_at
    BEFORE UPDATE ON collection
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE collection IS 'Подборка фильмов, например "Лучшее за 2025 год".';

CREATE TABLE collection_film (
    collection_id bigint      NOT NULL,
    film_id       bigint      NOT NULL,
    position      smallint    NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT collection_film_pkey PRIMARY KEY (collection_id, film_id),
    CONSTRAINT collection_film_position_key UNIQUE (collection_id, position)
        DEFERRABLE INITIALLY IMMEDIATE,
    CONSTRAINT collection_film_collection_id_fkey FOREIGN KEY (collection_id)
        REFERENCES collection (id) ON DELETE CASCADE,
    CONSTRAINT collection_film_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT collection_film_position_check CHECK (position > 0)
);

COMMENT ON TABLE collection_film IS 'Фильмы в подборке с порядком показа.';

CREATE TABLE notification (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id bigint      NOT NULL,
    message    text        NOT NULL,
    is_read    boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notification_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE CASCADE,
    CONSTRAINT notification_message_check CHECK (length(message) BETWEEN 1 AND 1000)
);

CREATE TRIGGER notification_set_updated_at
    BEFORE UPDATE ON notification
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE notification IS
    'Уведомление пользователю, доставляется по вебсокету. Личные данные, удаляются вместе с аккаунтом.';

CREATE TABLE film_discussion (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    film_id    bigint      NOT NULL,
    is_closed  boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_discussion_film_id_key UNIQUE (film_id),
    CONSTRAINT film_discussion_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE
);

CREATE TRIGGER film_discussion_set_updated_at
    BEFORE UPDATE ON film_discussion
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE film_discussion IS 'Комната обсуждения фильма, одна на фильм.';

CREATE TABLE discussion_message (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    discussion_id bigint      NOT NULL,
    account_id    bigint      NOT NULL,
    message_text  text        NOT NULL,
    deleted_at    timestamptz,
    deleted_by    bigint,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT discussion_message_discussion_id_fkey FOREIGN KEY (discussion_id)
        REFERENCES film_discussion (id) ON DELETE CASCADE,
    CONSTRAINT discussion_message_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT discussion_message_deleted_by_fkey FOREIGN KEY (deleted_by)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT discussion_message_message_text_check
        CHECK (length(message_text) BETWEEN 1 AND 4000),
    CONSTRAINT discussion_message_deleted_pair_check
        CHECK ((deleted_at IS NULL) = (deleted_by IS NULL))
);

CREATE TRIGGER discussion_message_set_updated_at
    BEFORE UPDATE ON discussion_message
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE discussion_message IS
    'Сообщение в обсуждении фильма. Отправляется и принимается по вебсокету.';
COMMENT ON COLUMN discussion_message.deleted_by IS
    'Кто удалил: если совпадает с account_id — сам автор, иначе модератор.';

CREATE TABLE moderation_task (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    review_id   bigint,
    message_id  bigint,
    status      text        NOT NULL DEFAULT 'pending',
    source      text        NOT NULL DEFAULT 'auto',
    model_name  text,
    model_score numeric(4, 3),
    decided_by  bigint,
    reason      text,
    decided_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT moderation_task_review_id_fkey FOREIGN KEY (review_id)
        REFERENCES review (id) ON DELETE CASCADE,
    CONSTRAINT moderation_task_message_id_fkey FOREIGN KEY (message_id)
        REFERENCES discussion_message (id) ON DELETE CASCADE,
    CONSTRAINT moderation_task_decided_by_fkey FOREIGN KEY (decided_by)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT moderation_task_target_check
        CHECK (num_nonnulls(review_id, message_id) = 1),
    CONSTRAINT moderation_task_status_check
        CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT moderation_task_source_check CHECK (source IN ('auto', 'manual')),
    CONSTRAINT moderation_task_model_score_check CHECK (model_score BETWEEN 0 AND 1),
    CONSTRAINT moderation_task_reason_check CHECK (length(reason) BETWEEN 3 AND 1000),
    CONSTRAINT moderation_task_decided_at_check
        CHECK ((status = 'pending') = (decided_at IS NULL)),
    CONSTRAINT moderation_task_auto_check
        CHECK (source <> 'auto' OR (model_name IS NOT NULL AND decided_by IS NULL)),
    CONSTRAINT moderation_task_manual_check
        CHECK (source <> 'manual' OR status = 'pending' OR decided_by IS NOT NULL),
    CONSTRAINT moderation_task_model_name_check
        CHECK (length(model_name) BETWEEN 2 AND 64)
);

CREATE TRIGGER moderation_task_set_updated_at
    BEFORE UPDATE ON moderation_task
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE UNIQUE INDEX moderation_task_review_pending_idx
    ON moderation_task (review_id) WHERE status = 'pending' AND review_id IS NOT NULL;
CREATE UNIQUE INDEX moderation_task_message_pending_idx
    ON moderation_task (message_id) WHERE status = 'pending' AND message_id IS NOT NULL;

COMMENT ON TABLE moderation_task IS
    'Проверка рецензии или сообщения: автоматическая (нейросеть) или ручная. Публикуется то, что получило approved.';
COMMENT ON COLUMN moderation_task.model_score IS
    'Уверенность модели от 0 до 1. По ней настраивается порог отправки на ручную проверку.';

CREATE TABLE account_ban (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id    bigint      NOT NULL,
    moderator_id  bigint      NOT NULL,
    reason        text        NOT NULL,
    active_during tstzrange   NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT account_ban_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT account_ban_moderator_id_fkey FOREIGN KEY (moderator_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT account_ban_self_check CHECK (account_id <> moderator_id),
    CONSTRAINT account_ban_reason_check CHECK (length(reason) BETWEEN 3 AND 1000),
    CONSTRAINT account_ban_active_during_check CHECK (NOT isempty(active_during)),
    CONSTRAINT account_ban_no_overlap
        EXCLUDE USING gist (account_id WITH =, active_during WITH &&)
);

CREATE TRIGGER account_ban_set_updated_at
    BEFORE UPDATE ON account_ban
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE account_ban IS
    'Блокировка пользователя на период. EXCLUDE запрещает пересекающиеся блокировки одного пользователя.';
COMMENT ON COLUMN account_ban.active_during IS
    'Период действия. Верхняя граница пустая — блокировка бессрочная.';

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

CREATE TABLE film_revision (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    film_id         bigint      NOT NULL,
    changed_by      bigint,
    title           text        NOT NULL,
    original_title  text,
    film_type       text        NOT NULL,
    production_year smallint    NOT NULL,
    duration_min    smallint,
    age_limit       smallint    NOT NULL,
    description     text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT film_revision_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_revision_changed_by_fkey FOREIGN KEY (changed_by)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT film_revision_title_check CHECK (length(title) BETWEEN 1 AND 255),
    CONSTRAINT film_revision_film_type_check
        CHECK (film_type IN ('movie', 'series')),
    CONSTRAINT film_revision_production_year_check
        CHECK (production_year BETWEEN 1888 AND 2200),
    CONSTRAINT film_revision_age_limit_check CHECK (age_limit IN (0, 6, 12, 16, 18))
);

COMMENT ON TABLE film_revision IS
    'Предыдущие версии карточки фильма. Пишется триггером при изменении film.';
COMMENT ON COLUMN film_revision.changed_by IS
    'Кто правил карточку. Берётся из параметра сессии app.account_id, который выставляет приложение.';

CREATE FUNCTION current_account_id() RETURNS bigint
    LANGUAGE plpgsql STABLE AS $$
DECLARE
    raw_value text := current_setting('app.account_id', true);
BEGIN
    IF raw_value IS NULL OR raw_value = '' THEN
        RETURN NULL;
    END IF;

    RETURN raw_value::bigint;
END;
$$;

COMMENT ON FUNCTION current_account_id() IS
    'Идентификатор пользователя, от имени которого выполняется запрос. Приложение задаёт его через SET LOCAL app.account_id.';

CREATE FUNCTION write_film_revision() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.title IS DISTINCT FROM NEW.title
       OR OLD.original_title IS DISTINCT FROM NEW.original_title
       OR OLD.film_type IS DISTINCT FROM NEW.film_type
       OR OLD.production_year IS DISTINCT FROM NEW.production_year
       OR OLD.duration_min IS DISTINCT FROM NEW.duration_min
       OR OLD.age_limit IS DISTINCT FROM NEW.age_limit
       OR OLD.description IS DISTINCT FROM NEW.description THEN
        INSERT INTO film_revision (film_id, changed_by, title, original_title,
                                   film_type, production_year, duration_min,
                                   age_limit, description, created_at)
        VALUES (OLD.id, current_account_id(), OLD.title, OLD.original_title,
                OLD.film_type, OLD.production_year, OLD.duration_min,
                OLD.age_limit, OLD.description, OLD.updated_at);
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER film_write_revision
    BEFORE UPDATE ON film
    FOR EACH ROW EXECUTE FUNCTION write_film_revision();

CREATE TABLE review_revision (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    review_id         bigint      NOT NULL,
    title             text        NOT NULL,
    content           text        NOT NULL,
    contains_spoilers boolean     NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT review_revision_review_id_fkey FOREIGN KEY (review_id)
        REFERENCES review (id) ON DELETE CASCADE,
    CONSTRAINT review_revision_title_check CHECK (length(title) BETWEEN 3 AND 255),
    CONSTRAINT review_revision_content_check
        CHECK (length(content) BETWEEN 50 AND 20000)
);

COMMENT ON TABLE review_revision IS
    'Предыдущие версии рецензии. Пишется триггером при изменении текста.';

CREATE FUNCTION write_review_revision() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.title IS DISTINCT FROM NEW.title
       OR OLD.content IS DISTINCT FROM NEW.content
       OR OLD.contains_spoilers IS DISTINCT FROM NEW.contains_spoilers THEN
        INSERT INTO review_revision (review_id, title, content,
                                     contains_spoilers, created_at)
        VALUES (OLD.id, OLD.title, OLD.content, OLD.contains_spoilers,
                OLD.updated_at);
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER review_write_revision
    BEFORE UPDATE ON review
    FOR EACH ROW EXECUTE FUNCTION write_review_revision();

CREATE TABLE account_role_history (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id  bigint      NOT NULL,
    old_role_id bigint,
    new_role_id bigint      NOT NULL,
    changed_by  bigint,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT account_role_history_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT account_role_history_old_role_id_fkey FOREIGN KEY (old_role_id)
        REFERENCES role (id) ON DELETE RESTRICT,
    CONSTRAINT account_role_history_new_role_id_fkey FOREIGN KEY (new_role_id)
        REFERENCES role (id) ON DELETE RESTRICT,
    CONSTRAINT account_role_history_changed_by_fkey FOREIGN KEY (changed_by)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT account_role_history_change_check
        CHECK (old_role_id IS NULL OR old_role_id <> new_role_id)
);

COMMENT ON TABLE account_role_history IS
    'История назначения ролей: когда и кем пользователю выдана роль. Пустая прежняя роль означает регистрацию.';

CREATE FUNCTION write_account_role_history() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    IF tg_op = 'INSERT' THEN
        INSERT INTO account_role_history (account_id, old_role_id, new_role_id,
                                          changed_by)
        VALUES (NEW.id, NULL, NEW.role_id, current_account_id());
    ELSIF NEW.role_id <> OLD.role_id THEN
        INSERT INTO account_role_history (account_id, old_role_id, new_role_id,
                                          changed_by)
        VALUES (NEW.id, OLD.role_id, NEW.role_id, current_account_id());
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER account_write_role_history
    AFTER INSERT OR UPDATE OF role_id ON account
    FOR EACH ROW EXECUTE FUNCTION write_account_role_history();

CREATE TABLE film_rating_daily (
    film_id       bigint         NOT NULL,
    stat_date     date           NOT NULL,
    rating_count  integer        NOT NULL,
    rating_sum    integer        NOT NULL,
    average_score numeric(4, 2)  NOT NULL,
    created_at    timestamptz    NOT NULL DEFAULT now(),
    CONSTRAINT film_rating_daily_pkey PRIMARY KEY (film_id, stat_date),
    CONSTRAINT film_rating_daily_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT film_rating_daily_rating_count_check CHECK (rating_count > 0),
    CONSTRAINT film_rating_daily_rating_sum_check CHECK (rating_sum > 0),
    CONSTRAINT film_rating_daily_average_score_check
        CHECK (average_score BETWEEN 1 AND 10),
    CONSTRAINT film_rating_daily_stat_date_check
        CHECK (stat_date >= DATE '1888-01-01')
);

COMMENT ON TABLE film_rating_daily IS
    'Снимок рейтинга фильма на конец дня: по нему строится график изменения оценки во времени.';
COMMENT ON COLUMN film_rating_daily.average_score IS
    'Среднее на дату снимка. Это не дублирование: текущее среднее считается по film_rating, а здесь хранится история.';

INSERT INTO account_role_history (account_id, old_role_id, new_role_id, created_at)
SELECT id, NULL, role_id, created_at
  FROM account
 WHERE NOT EXISTS (SELECT 1
                     FROM account_role_history AS h
                    WHERE h.account_id = account.id);

CREATE TABLE visitor (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id     text        NOT NULL,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT visitor_public_id_key UNIQUE (public_id),
    CONSTRAINT visitor_public_id_check CHECK (public_id ~ '^[A-Za-z0-9_-]{16,64}$')
);

CREATE TRIGGER visitor_set_updated_at
    BEFORE UPDATE ON visitor
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE visitor IS
    'Анонимный посетитель сайта. Идентификатор хранится в браузере и связывает события до и после входа в аккаунт.';

CREATE TABLE traffic_source (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    utm_source   text        NOT NULL,
    utm_medium   text,
    utm_campaign text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT traffic_source_key
        UNIQUE NULLS NOT DISTINCT (utm_source, utm_medium, utm_campaign),
    CONSTRAINT traffic_source_utm_source_check CHECK (length(utm_source) BETWEEN 1 AND 128),
    CONSTRAINT traffic_source_utm_medium_check CHECK (length(utm_medium) BETWEEN 1 AND 128),
    CONSTRAINT traffic_source_utm_campaign_check CHECK (length(utm_campaign) BETWEEN 1 AND 128)
);

CREATE TRIGGER traffic_source_set_updated_at
    BEFORE UPDATE ON traffic_source
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE traffic_source IS
    'Рекламный источник перехода: пара utm-меток. Справочник, чтобы не повторять метки в каждом визите.';

CREATE TABLE page (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    path       text        NOT NULL,
    title      text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT page_path_key UNIQUE (path),
    CONSTRAINT page_path_check CHECK (path ~ '^/' AND length(path) <= 512),
    CONSTRAINT page_title_check CHECK (length(title) BETWEEN 1 AND 255)
);

CREATE TRIGGER page_set_updated_at
    BEFORE UPDATE ON page
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE page IS
    'Страница сайта в виде шаблона пути, например /film/:id. Справочник для событий просмотра.';

CREATE TABLE event_type (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code       text        NOT NULL,
    title      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT event_type_code_key UNIQUE (code),
    CONSTRAINT event_type_title_key UNIQUE (title),
    CONSTRAINT event_type_code_check CHECK (code ~ '^[a-z_]{3,48}$'),
    CONSTRAINT event_type_title_check CHECK (length(title) BETWEEN 2 AND 128)
);

CREATE TRIGGER event_type_set_updated_at
    BEFORE UPDATE ON event_type
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE event_type IS
    'Справочник типов событий: просмотр страницы, клик, поиск, добавление в папку.';

CREATE TABLE visit (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    visitor_id        bigint      NOT NULL,
    account_id        bigint,
    traffic_source_id bigint,
    started_at        timestamptz NOT NULL DEFAULT now(),
    finished_at       timestamptz,
    user_agent        text,
    ip_address        inet,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT visit_visitor_id_fkey FOREIGN KEY (visitor_id)
        REFERENCES visitor (id) ON DELETE CASCADE,
    CONSTRAINT visit_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT visit_traffic_source_id_fkey FOREIGN KEY (traffic_source_id)
        REFERENCES traffic_source (id) ON DELETE RESTRICT,
    CONSTRAINT visit_user_agent_check CHECK (length(user_agent) BETWEEN 1 AND 512),
    CONSTRAINT visit_finished_at_check CHECK (finished_at >= started_at)
);

CREATE TRIGGER visit_set_updated_at
    BEFORE UPDATE ON visit
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE visit IS
    'Визит на сайт: цепочка событий одного посетителя. Пустой account_id означает, что пользователь не вошёл.';

CREATE TABLE analytic_event (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    visit_id      bigint      NOT NULL,
    event_type_id bigint      NOT NULL,
    page_id       bigint,
    film_id       bigint,
    person_id     bigint,
    collection_id bigint,
    element_code  text,
    occurred_at   timestamptz NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT analytic_event_visit_id_fkey FOREIGN KEY (visit_id)
        REFERENCES visit (id) ON DELETE CASCADE,
    CONSTRAINT analytic_event_event_type_id_fkey FOREIGN KEY (event_type_id)
        REFERENCES event_type (id) ON DELETE RESTRICT,
    CONSTRAINT analytic_event_page_id_fkey FOREIGN KEY (page_id)
        REFERENCES page (id) ON DELETE RESTRICT,
    CONSTRAINT analytic_event_film_id_fkey FOREIGN KEY (film_id)
        REFERENCES film (id) ON DELETE CASCADE,
    CONSTRAINT analytic_event_person_id_fkey FOREIGN KEY (person_id)
        REFERENCES person (id) ON DELETE CASCADE,
    CONSTRAINT analytic_event_collection_id_fkey FOREIGN KEY (collection_id)
        REFERENCES collection (id) ON DELETE CASCADE,
    CONSTRAINT analytic_event_object_check
        CHECK (num_nonnulls(film_id, person_id, collection_id) <= 1),
    CONSTRAINT analytic_event_element_code_check
        CHECK (element_code ~ '^[a-z0-9_-]{2,64}$'),
    CONSTRAINT analytic_event_occurred_at_check
        CHECK (occurred_at >= TIMESTAMPTZ '2020-01-01 00:00+00')
);

COMMENT ON TABLE analytic_event IS
    'Событие с фронта: просмотр страницы, клик по элементу, поиск. Строки не изменяются.';
COMMENT ON COLUMN analytic_event.element_code IS
    'Код элемента интерфейса, по которому кликнули, например poster_play или add_to_favorite.';
COMMENT ON COLUMN analytic_event.occurred_at IS
    'Момент события на стороне клиента; created_at -- момент записи на сервере.';

CREATE TABLE search_query (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    visit_id     bigint      NOT NULL,
    query_text   text        NOT NULL,
    result_count integer     NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT search_query_visit_id_fkey FOREIGN KEY (visit_id)
        REFERENCES visit (id) ON DELETE CASCADE,
    CONSTRAINT search_query_query_text_check CHECK (length(query_text) BETWEEN 1 AND 255),
    CONSTRAINT search_query_result_count_check CHECK (result_count >= 0)
);

COMMENT ON TABLE search_query IS
    'Поисковые запросы: текст запроса и число найденных результатов. Запросы без результатов показывают, чего не хватает в каталоге.';

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

-- Правила имени пользователя приводятся к правилам бэкенда и фронтенда:
-- буквы любого алфавита, цифры, пробел, дефис и подчёркивание, от 2 до 32 символов.
-- Набор символов проверяет бэкенд: классы символов в регулярных выражениях
-- PostgreSQL зависят от локали базы, поэтому в ограничении остаются длина
-- и отсутствие пробелов по краям.
-- Удаление и добавление в одной команде: если новое ограничение не встанет,
-- старое не пропадёт.
ALTER TABLE account
    DROP CONSTRAINT account_username_check,
    ADD CONSTRAINT account_username_check
        CHECK (length(username) BETWEEN 2 AND 32 AND username = btrim(username));

COMMENT ON CONSTRAINT account_username_check ON account IS
    'Имя от 2 до 32 символов без пробелов по краям. Допустимые символы проверяет бэкенд.';

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

ALTER TABLE review
    ADD COLUMN version integer NOT NULL DEFAULT 1,
    ADD CONSTRAINT review_version_check CHECK (version > 0);

COMMENT ON COLUMN review.version IS
    'Номер версии строки для оптимистической блокировки: клиент присылает версию, которую читал, и UPDATE проходит только если она совпала.';
