# OnlineConf backend integration

## Runtime contract

The product backend reads `GET /internal/onlineconf/snapshot` from the OnlineConf
container with `Authorization: Bearer <ONLINECONF_SERVICE_TOKEN>`. The endpoint is
available only on the Docker network and is not present in the public proxy config.

The adapter currently accepts this versioned snapshot shape:

```json
{
  "revision": 42,
  "configs": [
    {"scope": "backend", "key": "reviews.max_length", "value": 5000}
  ],
  "flags": [
    {
      "scope": "backend",
      "key": "reviews.write_enabled",
      "enabled": true,
      "rollout": 25,
      "included_user_ids": [101],
      "excluded_user_ids": [102]
    }
  ]
}
```

The rollout result is deterministic for `(scope, key, user_id)`. Exclusion takes
priority over inclusion; inclusion takes priority over percentage rollout. Guest
user `0` cannot write reviews.

## Cache and fallback

Startup performs one synchronous fetch with a two-second timeout. A single worker
then refreshes every ten seconds. Product requests read one immutable in-memory
snapshot and never call OnlineConf or its database.

Before the first valid snapshot the fallback is deliberately fail-closed:

- `reviews.write_enabled = false`;
- `reviews.max_length = 20000` (the existing domain maximum);
- `revision = 0`.

After a successful refresh, transport errors, invalid JSON, invalid values, and
older revisions preserve the last-known-good snapshot. A successful newer snapshot
replaces the complete state; a missing key therefore gets its per-key fallback and
does not retain a deleted value.

`reviews.max_length` must be a JSON integer from `50` through `20000`. Review create
and update enforce the flag and dynamic code-point limit on the server. Update keeps
the existing owner and optimistic-version checks.

## Public runtime endpoint

`GET /api/onlineconf/runtime` returns only the public projection for the current
session: revision, `reviews.max_length`, and the computed
`reviews.write_enabled`. A caller-supplied user id is ignored. The response is
`Cache-Control: private, no-store` and `Vary: Cookie`; service tokens, override
lists, internal scopes, and raw snapshot data are never serialized.

No session cookie means the guest projection. An invalid supplied session cookie
returns `401`, matching the existing authentication behavior. The route remains
inside the existing CSRF, security-header, and cookie configuration.

## Deployment and existing databases

Copy `.env.example` to `.env`, replace every secret, build or pull the image named
by `ONLINECONF_IMAGE`, and start the stack. The OnlineConf port is bound only to
`127.0.0.1`; the reverse proxy publishes only `/onlineconf/` and
`/api/onlineconf/admin/`. Use the provided nginx location template in the existing
TLS server block. Do not add a public proxy for `/internal/`.

PostgreSQL executes `docker-entrypoint-initdb.d` only for a new empty data volume.
Do not remove `db_data` to force a migration. Once the OnlineConf owner supplies an
idempotent up migration, apply it explicitly to the existing database:

```powershell
.\scripts\apply-onlineconf-migration.ps1 `
  -MigrationFile .\db\migrations\000016_onlineconf.up.sql
```

The script checks that the running database contains the existing product schema,
uses `ON_ERROR_STOP`, and never drops the volume or product data. Take a database
backup before production migration and verify the OnlineConf healthcheck before
putting the admin routes into service.
