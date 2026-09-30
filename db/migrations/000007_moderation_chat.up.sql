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
