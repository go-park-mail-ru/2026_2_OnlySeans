# OnlineConf backend integration

## Runtime contract

The product backend reads `GET /internal/onlineconf/snapshot` from the OnlineConf
container with `Authorization: Bearer <ONLINECONF_SERVICE_TOKEN>`. The endpoint is
available only on the Docker network and is not present in the public proxy config.

The adapter decodes the shared `domain/onlineconf.SnapshotPayload`, calls its
`Validate()` method, and uses `onlineconfevaluator.Evaluate()` for every flag.
There is one wire contract and one evaluation algorithm for admin and product.

The snapshot shape is:

```json
{
  "revision": 42,
  "generated_at": "2026-11-28T09:00:00Z",
  "configs": [
    {"scope": "reviews", "key": "max_length", "value": 5000}
  ],
  "flags": [
    {
      "id": 1,
      "scope": "reviews",
      "key": "write_enabled",
      "enabled": true,
      "rollout_percent": 25,
      "salt": "stable-example",
      "force_on_user_ids": [101],
      "force_off_user_ids": [102]
    }
  ]
}
```

The global switch takes precedence over all overrides. Exclusion takes priority
over inclusion, followed by rollout. The rollout bucket uses the first eight
SHA-256 bytes of `salt:user_id` as a big-endian integer modulo 10000, and matches
when the bucket is below `rollout_percent * 100`. Guest
user `0` cannot write reviews.

## Cache and fallback

Startup performs one synchronous fetch with a two-second timeout. A single worker
then refreshes every ten seconds. The maximum configured interval is 15 seconds
and the maximum client timeout is 2 seconds. Product requests read one immutable in-memory
snapshot and never call OnlineConf or its database.

Before the first valid snapshot the fallback is deliberately fail-closed:

- `reviews.write_enabled = false`;
- `reviews.max_length = 20000` (the existing domain maximum);
- `revision = 0`;
- `snapshot_ready = false`.

The first accepted snapshot sets `snapshot_ready = true`, even at revision zero.
It remains true while the cache serves the last successful snapshot.

After a successful refresh, transport errors, invalid JSON, invalid values, and
older revisions preserve the last-known-good snapshot. A successful newer snapshot
replaces the complete state; a missing key therefore gets its per-key fallback and
does not retain a deleted value.

`reviews.max_length` must be a JSON integer from `50` through `20000`. Review create
and update enforce the flag and dynamic code-point limit on the server. Update keeps
the existing owner and optimistic-version checks.

## Public runtime endpoint

`GET /api/onlineconf/runtime` returns only the public projection for the current
session: revision, snapshot readiness, `reviews.max_length`, `reviews.write_enabled`,
and all configuration keys and flags from the explicitly public `frontend` scope.
All flags are evaluated for the current session from the same cached revision.
Strings, objects, arrays, booleans, numbers, and null retain their JSON types.
Settings from other scopes stay private, apart from the two explicit review keys.
Treat the entire `frontend` scope as browser-visible: keep secrets in private scopes.
The existing `frontend.home_blocks` and `frontend.home_blocks_enabled` remain supported. Home blocks are an ordered JSON
array. The admin API accepts up to six blocks. A `collection` needs a positive,
JavaScript-safe integer `id`. A `banner` needs a `title` (up to 100 UTF-16 code
units after trimming) and can include `text` (up to 300) and an `action` with a
`label` (up to 40) and an internal product `href`. Invalid writes return HTTP
400. Older values remain readable so admins can correct them.
An excluded user, a missing config or a non-array value receives an empty array.
A caller-supplied user id is ignored. The response is
`Cache-Control: private, no-store` and `Vary: Cookie`; service tokens, override
lists, internal scopes, and raw snapshot data are never serialized.

No session cookie means the guest projection. An invalid supplied session cookie
returns `401`, matching the existing authentication behavior. The route remains
inside the existing CSRF, security-header, and cookie configuration.

### Examples for the product frontend

Runtime response (HTTP 200):

```json
{"revision":42,"snapshot_ready":true,"configs":{"reviews.max_length":500,"frontend.home_blocks":[{"type":"collection","id":1}]},"flags":{"reviews.write_enabled":true,"frontend.home_blocks_enabled":true}}
```

Writing with the flag disabled (HTTP 403):

```json
{"error":"feature disabled","code":"feature_disabled","details":{"revision":42}}
```

Writing a review above the dynamic limit (HTTP 400):

```json
{"error":"review content is too long","code":"review_too_long","details":{"max_length":100,"revision":42}}
```

## Deployment and existing databases

After integrating the OnlineConf service sources and migration from PR #42,
copy `.env.example` to `.env`, replace every secret, and run
`docker compose up -d --build`. Compose builds the separate `onlineconf` target
from `./cmd/onlineconf`. It checks `/healthz`; the product backend can start
independently and adopts the snapshot when its background poll succeeds. The
OnlineConf host port is bound to `127.0.0.1`. Install `deploy/nginx/onlineconf.locations.conf` as
`/etc/nginx/snippets/onlineconf.locations.conf`, then add the following line to
the existing domain's TLS `server` block (remove overlapping locations first):

```nginx
include /etc/nginx/snippets/onlineconf.locations.conf;
```

Run `nginx -t` before reloading nginx. Keep the explicit `/internal/` rejection.
This include uses host ports 8080 and 8081; if `ONLINECONF_HOST_PORT` differs,
update the admin proxy upstream accordingly.

Build the separate administrative SPA in its frontend repository. Install its
artifacts with:

```bash
bash scripts/install-onlineconf-spa.sh /path/to/admin/dist
```

The installer requires `onlineconf/index.html` and `assets/onlineconf/`, copies
assets first and replaces HTML atomically. It preserves previous asset files so
open tabs keep working. It never copies the product SPA over the admin entry point.
Keep nginx's `mime.types` include enabled. The template serves nested SPA routes
such as `/onlineconf/flags` and returns 404 for missing bundles.

PostgreSQL executes `docker-entrypoint-initdb.d` only for a new empty data volume.
Do not remove `db_data` to force a migration. For an existing database, run the OnlineConf migration before bringing
the service online. Start the database with `docker compose up -d db`. From the
repository root, pass the values of `DB_USER` and `DB_NAME` from your Compose
`.env`. The example uses the values in `.env.example`:

```powershell
.\scripts\apply-onlineconf-migration.ps1 `
  -MigrationFile .\db\migrations\000016_onlineconf.up.sql `
  -DatabaseUser 'postgres' `
  -DatabaseName 'onlyseans'
```

You can omit the two database arguments if `DB_USER` and `DB_NAME` are already
exported in your PowerShell session. Docker Compose reads its own `.env` file;
that file does not set variables in the calling PowerShell process.

The script uses one transaction and a database lock, skips a complete
000016 schema, and rejects a partially installed schema. Take a database
backup before production migration and verify the OnlineConf healthcheck before
putting the admin routes into service.

### Delegating OnlineConf access

For an existing database at migration 000016, apply the next migration once
before starting the updated OnlineConf service:

```sh
psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -1 -f db/migrations/000017_onlineconf_admins.up.sql
```

Register a separate Frame account and verify its ID and username. To bootstrap
the first OnlineConf superadmin, run the following in psql with `account_id` and
`username` set to that verified account. Check that it returns one row.

```sql
INSERT INTO oc_admin_access (account_id, access_role)
SELECT id, 'superadmin' FROM account
WHERE id = :'account_id' AND username = :'username' AND deleted_at IS NULL
RETURNING account_id, access_role;
```

The superadmin signs in through Frame and opens `/onlineconf/administrators` to
find an existing account by its exact username and grant admin access. The grant
records the actor and timestamp. It takes effect on the recipient's next admin
request, including in an existing session. Frame roles stay intact. Existing
Frame admins retain OnlineConf access. Only OnlineConf superadmins can delegate. Apply migration 000018 after 000017 to
enable explicit access revocation and the audit index. The administrators page
lists effective access (including legacy Frame admins), allows role changes and
keeps at least one active superadmin. The audit API accepts `actor_id` together
with the resource filters and lists historical authors at `/admin/audit/authors`
under the OnlineConf API prefix.

The public feature flag `frontend.library_add_enabled` controls additions to all
three personal lists. A ready snapshot with no flag preserves the original
enabled behavior. The product backend checks the same user's cached flag before
saving. Reads and removals continue while adding is disabled.

## Generic backend settings

Inject `onlineconf.SettingsProvider` into use cases that need arbitrary settings:

```go
value, found := settings.GetConfig("payments", "checkout_options")
decision := settings.EvaluateFlag("payments", "new_checkout", userID)
```

Both methods read only local memory. `GetConfig` returns an owned copy of JSON;
`EvaluateFlag` returns the shared decision, reason and bucket. Missing config keys
return `found=false`; missing flags and the cold fallback are disabled.
For several settings that must share a revision, take `cache.SettingsSnapshot()`
once and call its `GetConfig`, `EvaluateFlag`, and `Revision` methods. That view
remains unchanged when a newer revision is published.

A new frontend key, for example `frontend/button_color`, is exposed automatically
as `configs["frontend.button_color"]`. The product frontend must still read it and
apply the color. Creating an arbitrary key cannot add new product behavior by itself.

## Verification

Run tests from `tests/`, including the real PostgreSQL admin-to-product workflow:

```bash
ONLINECONF_TEST_DATABASE_URL='postgres://...' go test -race ./tests/onlineconf ./tests/onlineconfruntime ./tests/reviews ./tests/router ./tests/config
```

After deploying the SPA and nginx include, check the real origin:

```bash
curl -i https://your-domain/onlineconf/flags
curl -i https://your-domain/assets/onlineconf/nonexistent.js
curl -i https://your-domain/api/onlineconf/runtime
curl -i https://your-domain/api/onlineconf/admin/me
curl -i https://your-domain/internal/onlineconf/snapshot
```

Expect admin HTML for the nested SPA route, 404 for a missing asset,
`private, no-store` for runtime, 401 for an unauthenticated admin request, and
404 for the internal API. Verify a regular user's cookie gets 403 on admin API.
Finally change the review limit and flag in admin, then check runtime and a direct
review write for the same account within one minute.

## Scheduled rollouts and delegated access

Apply migration 000019 after 000018 before starting the updated OnlineConf service.
Existing flags keep their current audience. New flags start at zero.
Use `/onlineconf/rollouts` to choose a policy, schedule the next stage and approve it.
The service consumes each approval once and checks the approver's current rights at execution.
Stopping a schedule preserves current coverage. Editing a flag cancels its pending stage.
Every applied stage publishes a new snapshot and records the approving account in the flag history.

Use `/OnlyAdmin/access` as a superadmin to grant an operation on a scope or key to a user or group.
An ordinary Frame account with an effective grant gets scoped access automatically.
Explicit no-access takes precedence. Change a full administrator to the scoped role to restrict them.

For example, grant a group `read` on `config` with scope `reviews` and key `max_length`.
A member can list and open that parameter. Other keys, flags with the same key, and the
scope's own title and description require separate grants. Use kind `scope` with action
`read` to open that scope under **Доступные области**. Use kind `*` to grant the
chosen action across resource types within that scope. An empty key covers all keys of the selected type.

Grant `create`, `update`, `delete` and flag rollout `approve` independently. A successful
create or update without a read grant returns an ID and version. A scoped account can inspect its
own effective grants at `GET /api/onlineconf/admin/governance/my-grants`. Changes to
membership and grants take effect on subsequent requests in the current session.
