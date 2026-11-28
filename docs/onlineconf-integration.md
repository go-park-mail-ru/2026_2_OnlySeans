# OnlineConf backend integration

## Runtime contract

The product backend reads `GET /internal/onlineconf/snapshot` from the OnlineConf
container with `Authorization: Bearer <ONLINECONF_SERVICE_TOKEN>`. The endpoint is
available only on the Docker network and is not present in the public proxy config.

The adapter currently accepts this versioned snapshot shape:

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
`frontend.home_blocks` and `frontend.home_blocks_enabled`. Both flags are evaluated
for the current session from the same cached revision. Home blocks are an ordered JSON
array. An excluded user, a missing config or a non-array value receives an empty array.
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
OnlineConf host port is bound to `127.0.0.1`. Use the nginx location template in
the existing TLS server block and keep the explicit `/internal/` rejection.

Copy the administrative SPA's `dist/` contents into
`/var/www/onlyseans-onlineconf/`. This places the HTML at
`onlineconf/index.html` and bundles under `assets/onlineconf/` inside that root.
Keep nginx's `mime.types` include enabled. The template serves nested SPA routes
such as `/onlineconf/flags` and returns 404 for missing bundles.

PostgreSQL executes `docker-entrypoint-initdb.d` only for a new empty data volume.
Do not remove `db_data` to force a migration. For an existing database, run the OnlineConf migration before bringing
the service online:

```powershell
.\scripts\apply-onlineconf-migration.ps1 `
  -MigrationFile .\db\migrations\000016_onlineconf.up.sql
```

The script uses one transaction and a database lock, skips a complete
000016 schema, and rejects a partially installed schema. Take a database
backup before production migration and verify the OnlineConf healthcheck before
putting the admin routes into service.
