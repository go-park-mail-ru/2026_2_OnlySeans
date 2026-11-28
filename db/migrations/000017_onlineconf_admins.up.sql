CREATE TABLE oc_admin_access (
    account_id bigint PRIMARY KEY REFERENCES account(id) ON DELETE CASCADE,
    access_role text NOT NULL CHECK (access_role IN ('admin', 'superadmin')),
    granted_by bigint REFERENCES account(id) ON DELETE SET NULL,
    granted_at timestamptz NOT NULL DEFAULT now()
);
