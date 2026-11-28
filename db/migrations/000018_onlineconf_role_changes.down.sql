-- Resolve explicit access denials before rolling back to the two-role schema.
ALTER TABLE oc_admin_access DROP CONSTRAINT oc_admin_access_access_role_check,
    ADD CONSTRAINT oc_admin_access_access_role_check CHECK (access_role IN ('admin', 'superadmin'));
DROP INDEX oc_audit_actor_created_idx;
