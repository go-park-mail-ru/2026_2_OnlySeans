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
