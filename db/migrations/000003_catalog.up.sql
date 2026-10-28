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
