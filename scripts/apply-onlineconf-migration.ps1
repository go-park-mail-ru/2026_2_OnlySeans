param(
    [Parameter(Mandatory = $true)]
    [ValidateScript({ Test-Path -LiteralPath $_ -PathType Leaf })]
    [string]$MigrationFile,

    [string]$DatabaseUser = $env:DB_USER,
    [string]$DatabaseName = $env:DB_NAME
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($DatabaseUser) -or [string]::IsNullOrWhiteSpace($DatabaseName)) {
    throw 'Set DB_USER and DB_NAME or pass -DatabaseUser and -DatabaseName.'
}

$resolvedMigration = (Resolve-Path -LiteralPath $MigrationFile).Path
$composeServices = docker compose ps --services --status running
if ($LASTEXITCODE -ne 0 -or $composeServices -notcontains 'db') {
    throw 'The docker compose db service must be running.'
}

# This guard and the migration execute in a single database transaction.
$guard = @'
SELECT pg_advisory_xact_lock(160016);

DO $check$
DECLARE
    table_count integer;
BEGIN
    IF to_regclass('public.account') IS NULL THEN
        RAISE EXCEPTION 'Product schema is missing';
    END IF;

    SELECT count(*) INTO table_count
    FROM pg_tables
    WHERE schemaname = 'public'
      AND tablename IN ('oc_scopes', 'oc_configs', 'oc_flags',
                        'oc_flag_overrides', 'oc_state', 'oc_audit');

    IF table_count NOT IN (0, 6) THEN
        RAISE EXCEPTION 'Incomplete OnlineConf schema. Inspect the installed schema before retrying';
    END IF;

    IF table_count = 6 THEN
        IF NOT EXISTS (SELECT FROM public.oc_state WHERE id = 1) THEN
            RAISE EXCEPTION 'OnlineConf state row is missing';
        END IF;
    END IF;
END
$check$;

SELECT to_regclass('public.oc_state') IS NULL AS apply_000016 \gset
\if :apply_000016
'@

$end = @'
\echo Applied OnlineConf migration 000016.
\else
\echo OnlineConf migration 000016 is already installed.
\endif
'@

if ([IO.Path]::GetFileName($resolvedMigration) -ne '000016_onlineconf.up.sql') {
    throw 'Expected the OnlineConf migration 000016_onlineconf.up.sql.'
}

$migration = Get-Content -LiteralPath $resolvedMigration -Raw
($guard, $migration, $end -join "`n") | docker compose exec -T db psql `
    --single-transaction `
    -v ON_ERROR_STOP=1 `
    -U $DatabaseUser `
    -d $DatabaseName `
    -f -
if ($LASTEXITCODE -ne 0) {
    throw 'OnlineConf migration 000016 failed.'
}
