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
    CONSTRAINT review_account_id_film_id_key UNIQUE (account_id, film_id),
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

CREATE TRIGGER review_set_updated_at
    BEFORE UPDATE ON review
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE review IS
    'Рецензия пользователя на фильм, одна на пару пользователь-фильм. Публикуется после модерации.';
COMMENT ON COLUMN review.deleted_by IS
    'Кто удалил рецензию: если совпадает с account_id — сам автор, иначе модератор.';
