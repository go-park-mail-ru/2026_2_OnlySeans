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
