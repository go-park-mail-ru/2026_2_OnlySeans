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

# docker-entrypoint-initdb.d is intentionally not used here: PostgreSQL only runs it
# while creating a fresh data directory, so it cannot migrate an existing volume.
$existingSchema = docker compose exec -T db psql `
    -v ON_ERROR_STOP=1 `
    -U $DatabaseUser `
    -d $DatabaseName `
    -Atc "SELECT to_regclass('public.account') IS NOT NULL"
$schemaResult = [string]($existingSchema | Select-Object -Last 1)
if ($LASTEXITCODE -ne 0 -or $schemaResult.Trim() -ne 't') {
    throw 'Existing product schema was not found; refusing to apply the migration.'
}

Get-Content -LiteralPath $resolvedMigration -Raw | docker compose exec -T db psql `
    --single-transaction `
    -v ON_ERROR_STOP=1 `
    -U $DatabaseUser `
    -d $DatabaseName `
    -f -
if ($LASTEXITCODE -ne 0) {
    throw 'OnlineConf migration failed. Existing data was not deleted.'
}

Write-Host "Applied migration to existing database: $resolvedMigration"
