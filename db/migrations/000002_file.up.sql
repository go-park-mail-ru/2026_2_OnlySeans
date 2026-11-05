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
