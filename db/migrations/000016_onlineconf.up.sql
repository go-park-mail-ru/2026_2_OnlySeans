CREATE TABLE oc_scopes (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        text        NOT NULL,
    title       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    version     bigint      NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oc_scopes_name_key UNIQUE (name),
    CONSTRAINT oc_scopes_name_check CHECK (name ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT oc_scopes_title_check CHECK (length(btrim(title)) > 0),
    CONSTRAINT oc_scopes_version_check CHECK (version > 0)
);

CREATE TRIGGER oc_scopes_set_updated_at
    BEFORE UPDATE ON oc_scopes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE oc_configs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scope_id    bigint      NOT NULL,
    key         text        NOT NULL,
    title       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    value       jsonb       NOT NULL,
    version     bigint      NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oc_configs_scope_id_fkey FOREIGN KEY (scope_id)
        REFERENCES oc_scopes (id) ON DELETE RESTRICT,
    CONSTRAINT oc_configs_scope_key_key UNIQUE (scope_id, key),
    CONSTRAINT oc_configs_key_check CHECK (key ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT oc_configs_title_check CHECK (length(btrim(title)) > 0),
    CONSTRAINT oc_configs_version_check CHECK (version > 0)
);

CREATE TRIGGER oc_configs_set_updated_at
    BEFORE UPDATE ON oc_configs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE oc_flags (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scope_id        bigint      NOT NULL,
    key             text        NOT NULL,
    title           text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    enabled         boolean     NOT NULL DEFAULT false,
    rollout_percent smallint    NOT NULL DEFAULT 0,
    salt            text        NOT NULL,
    version         bigint      NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oc_flags_scope_id_fkey FOREIGN KEY (scope_id)
        REFERENCES oc_scopes (id) ON DELETE RESTRICT,
    CONSTRAINT oc_flags_scope_key_key UNIQUE (scope_id, key),
    CONSTRAINT oc_flags_key_check CHECK (key ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT oc_flags_title_check CHECK (length(btrim(title)) > 0),
    CONSTRAINT oc_flags_rollout_percent_check CHECK (rollout_percent BETWEEN 0 AND 100),
    CONSTRAINT oc_flags_salt_check CHECK (length(btrim(salt)) > 0),
    CONSTRAINT oc_flags_version_check CHECK (version > 0)
);

CREATE TRIGGER oc_flags_set_updated_at
    BEFORE UPDATE ON oc_flags
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE oc_flag_overrides (
    flag_id    bigint      NOT NULL,
    account_id bigint      NOT NULL,
    enabled    boolean     NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oc_flag_overrides_pkey PRIMARY KEY (flag_id, account_id),
    CONSTRAINT oc_flag_overrides_flag_id_fkey FOREIGN KEY (flag_id)
        REFERENCES oc_flags (id) ON DELETE CASCADE,
    CONSTRAINT oc_flag_overrides_account_id_fkey FOREIGN KEY (account_id)
        REFERENCES account (id) ON DELETE RESTRICT
);

CREATE TABLE oc_state (
    id           smallint    PRIMARY KEY DEFAULT 1,
    revision     bigint      NOT NULL DEFAULT 0,
    snapshot     jsonb       NOT NULL,
    generated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oc_state_singleton_check CHECK (id = 1),
    CONSTRAINT oc_state_revision_check CHECK (revision >= 0),
    CONSTRAINT oc_state_snapshot_check CHECK (
        jsonb_typeof(snapshot) = 'object'
        AND snapshot ?& ARRAY['revision', 'generated_at', 'configs', 'flags']
        AND snapshot -> 'revision' = to_jsonb(revision)
        AND jsonb_typeof(snapshot -> 'generated_at') = 'string'
        AND jsonb_typeof(snapshot -> 'configs') = 'array'
        AND jsonb_typeof(snapshot -> 'flags') = 'array'
    )
);

INSERT INTO oc_state (id, revision, snapshot, generated_at)
VALUES (1, 0, jsonb_build_object(
    'revision', 0,
    'generated_at', now(),
    'configs', '[]'::jsonb,
    'flags', '[]'::jsonb
), now());

CREATE TABLE oc_audit (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id      bigint      NOT NULL,
    actor_name    text        NOT NULL,
    resource_type text        NOT NULL,
    resource_id   bigint      NOT NULL,
    scope         text        NOT NULL,
    key           text,
    action        text        NOT NULL,
    before        jsonb       NOT NULL DEFAULT 'null'::jsonb,
    after         jsonb       NOT NULL DEFAULT 'null'::jsonb,
    revision      bigint      NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oc_audit_actor_id_fkey FOREIGN KEY (actor_id)
        REFERENCES account (id) ON DELETE RESTRICT,
    CONSTRAINT oc_audit_actor_name_check CHECK (length(btrim(actor_name)) > 0),
    CONSTRAINT oc_audit_resource_type_check CHECK (resource_type IN ('scope', 'config', 'flag')),
    CONSTRAINT oc_audit_resource_id_check CHECK (resource_id > 0),
    CONSTRAINT oc_audit_scope_check CHECK (scope ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT oc_audit_key_check CHECK (
        (resource_type = 'scope' AND key IS NULL)
        OR (resource_type IN ('config', 'flag') AND key IS NOT NULL
            AND key ~ '^[a-z][a-z0-9_]{0,63}$')
    ),
    CONSTRAINT oc_audit_action_check CHECK (action IN ('create', 'update', 'delete')),
    CONSTRAINT oc_audit_revision_check CHECK (revision > 0),
    CONSTRAINT oc_audit_changes_check CHECK (
        (action = 'create' AND before = 'null'::jsonb AND jsonb_typeof(after) = 'object')
        OR (action = 'update' AND jsonb_typeof(before) = 'object' AND jsonb_typeof(after) = 'object')
        OR (action = 'delete' AND jsonb_typeof(before) = 'object' AND after = 'null'::jsonb)
    )
);

CREATE INDEX oc_audit_resource_created_at_idx
    ON oc_audit (resource_type, resource_id, created_at DESC, id DESC);

CREATE INDEX oc_audit_created_at_idx
    ON oc_audit (created_at DESC, id DESC);
