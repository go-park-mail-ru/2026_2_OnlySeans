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
