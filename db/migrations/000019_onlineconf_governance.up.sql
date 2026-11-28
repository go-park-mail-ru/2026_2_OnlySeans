ALTER TABLE oc_admin_access DROP CONSTRAINT oc_admin_access_access_role_check,
 ADD CHECK (access_role IN ('admin','superadmin','scoped','no_access'));
CREATE TABLE oc_groups(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name text UNIQUE NOT NULL CHECK(length(btrim(name)) BETWEEN 1 AND 120));
CREATE TABLE oc_members(group_id bigint REFERENCES oc_groups ON DELETE CASCADE, account_id bigint REFERENCES account ON DELETE CASCADE, PRIMARY KEY(group_id,account_id));
CREATE TABLE oc_grants(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 account_id bigint REFERENCES account ON DELETE CASCADE, group_id bigint REFERENCES oc_groups ON DELETE CASCADE,
 scope text NOT NULL CHECK(scope ~ '^[a-z][a-z0-9_]{0,63}$'), kind text NOT NULL CHECK(kind IN ('*','config','flag','scope')),
 key text NOT NULL DEFAULT '' CHECK(key='' OR key ~ '^[a-z][a-z0-9_]{0,63}$'),
 action text NOT NULL CHECK(action IN ('create','read','update','delete','approve')),
 CHECK(num_nonnulls(account_id,group_id)=1), CHECK(kind IN ('config','flag') OR key=''),
 UNIQUE NULLS NOT DISTINCT(account_id,group_id,scope,kind,key,action));
CREATE VIEW oc_effective_grants AS
 SELECT account_id,scope,kind,key,action FROM oc_grants WHERE account_id IS NOT NULL
 UNION ALL SELECT m.account_id,g.scope,g.kind,g.key,g.action FROM oc_grants g JOIN oc_members m USING(group_id);
CREATE FUNCTION oc_permitted(uid bigint, area text, resource text, resource_key text, verb text)
 RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT FROM oc_effective_grants g WHERE g.account_id=uid AND g.scope=area
 AND (g.kind='*' OR g.kind=resource) AND (g.key='' OR g.key=resource_key) AND g.action=verb) $$;

CREATE FUNCTION oc_valid_steps(steps int[]) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT COALESCE(cardinality(steps) BETWEEN 2 AND 100 AND steps[cardinality(steps)]=100 AND
 NOT EXISTS(SELECT FROM generate_subscripts(steps,1) n WHERE steps[n] IS NULL OR steps[n]<=COALESCE(steps[n-1],0) OR steps[n]>100),false) $$;
CREATE TABLE oc_policies(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,name text UNIQUE NOT NULL CHECK(length(btrim(name)) BETWEEN 1 AND 120),
 steps int[] NOT NULL CHECK(oc_valid_steps(steps)), wait_seconds integer NOT NULL CHECK(wait_seconds BETWEEN 1 AND 2592000));
INSERT INTO oc_policies(name,steps,wait_seconds) VALUES ('Стандартный',ARRAY[5,25,50,100],60);
CREATE TABLE oc_rollouts(flag_id bigint PRIMARY KEY REFERENCES oc_flags ON DELETE CASCADE,policy_id bigint NOT NULL DEFAULT 1 REFERENCES oc_policies,
 not_before timestamptz, due_at timestamptz, approved_by bigint REFERENCES account ON DELETE SET NULL, generation bigint NOT NULL DEFAULT 1);
INSERT INTO oc_rollouts(flag_id) SELECT id FROM oc_flags;
CREATE FUNCTION oc_guard_rollout() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='INSERT' THEN
   IF NEW.rollout_percent<>0 THEN RAISE EXCEPTION USING ERRCODE='P4101', MESSAGE='Начните раскатку с 0%.'; END IF;
 ELSIF (NEW.rollout_percent>OLD.rollout_percent OR (NEW.enabled AND NOT OLD.enabled AND NEW.rollout_percent>0))
 AND current_setting('onlineconf.approved_flag',true) IS DISTINCT FROM NEW.id::text THEN
   RAISE EXCEPTION USING ERRCODE='P4101', MESSAGE='Увеличьте процент через подтверждение этапа раскатки.';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER oc_guard BEFORE INSERT OR UPDATE ON oc_flags FOR EACH ROW EXECUTE FUNCTION oc_guard_rollout();
CREATE FUNCTION oc_sync_rollout() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='INSERT' THEN INSERT INTO oc_rollouts(flag_id) VALUES(NEW.id);
 ELSIF current_setting('onlineconf.approved_flag',true) IS DISTINCT FROM NEW.id::text THEN
   UPDATE oc_rollouts SET approved_by=NULL,due_at=NULL,generation=generation+1 WHERE flag_id=NEW.id;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER oc_sync AFTER INSERT OR UPDATE ON oc_flags FOR EACH ROW EXECUTE FUNCTION oc_sync_rollout();
CREATE INDEX oc_rollout_due ON oc_rollouts(due_at) WHERE approved_by IS NOT NULL;
