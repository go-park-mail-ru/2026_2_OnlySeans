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
