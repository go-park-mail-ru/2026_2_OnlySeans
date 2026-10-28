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
