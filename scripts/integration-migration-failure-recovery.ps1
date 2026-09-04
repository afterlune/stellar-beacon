[CmdletBinding()]
param(
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if (-not $AllowWrites) {
    throw 'Refusing to modify the isolated database. Re-run with -AllowWrites during an approved integration window.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')

# The test creates and removes a random schema in the fixed integration
# database. Ignore any local override so a stale compose file cannot redirect
# the DSN or container selection to another environment.
$env:INTEGRATION_USE_BASE_COMPOSE = '1'
Import-IntegrationEnv

$services = @(Invoke-IntegrationCompose -Arguments @('config', '--services'))
if ('postgresql' -notin $services) {
    throw 'The selected Compose project does not expose the isolated postgresql service.'
}
$statusLines = @(Invoke-IntegrationCompose -Arguments @('ps', '--format', 'json'))
$postgresRecords = @()
foreach ($line in $statusLines) {
    if ([string]::IsNullOrWhiteSpace([string]$line)) {
        continue
    }
    $postgresRecords += [string]$line | ConvertFrom-Json
}
$postgres = @($postgresRecords | Where-Object { [string]$_.Service -eq 'postgresql' })
if ($postgres.Count -ne 1 -or [string]$postgres[0].State -notin @('running', 'Up')) {
    throw 'The isolated postgresql container is not running; no failure recovery test was attempted.'
}
if ([string]$postgres[0].Name -notlike 'benetnasch-integration-*') {
    throw 'The selected postgresql container is outside the benetnasch-integration project.'
}

if ([string]::IsNullOrWhiteSpace($env:POSTGRES_PASSWORD)) {
    throw 'POSTGRES_PASSWORD is required in .env.integration.'
}
$encodedPassword = [uri]::EscapeDataString($env:POSTGRES_PASSWORD)
$env:MIGRATION_INTEGRATION_DSN = "postgres://postgres:$encodedPassword@127.0.0.1:15432/benetnasch?sslmode=disable"

& go test -tags=integration ./app/infra/persistence/migration -run '^TestRunnerFailureRollsBackMigrationAndRecoveryApplies$' -count=1
if ($LASTEXITCODE -ne 0) {
    throw 'Isolated migration failure recovery verification failed.'
}

Write-Host 'Isolated migration failure rollback and recovery verification passed.' -ForegroundColor Green
