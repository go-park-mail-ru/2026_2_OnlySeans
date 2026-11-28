ALTER TABLE oc_admin_access DROP CONSTRAINT oc_admin_access_access_role_check,
    ADD CONSTRAINT oc_admin_access_access_role_check CHECK (access_role IN ('admin', 'superadmin', 'no_access'));
CREATE INDEX oc_audit_actor_created_idx ON oc_audit(actor_id, created_at DESC, id DESC);
