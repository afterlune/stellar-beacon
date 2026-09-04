param(
    [switch]$AllowWrites,
    [switch]$VerifyIdempotency
)

$ErrorActionPreference = 'Stop'

if (-not $AllowWrites) {
    throw 'Refusing to modify the isolated database. Re-run with -AllowWrites during an approved integration window.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv
Assert-IntegrationMigrationStatusCommand

# This is an explicit operation for the isolated benetnasch-integration
# project. It never targets the production Compose project or its database.
Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'backend',
    '/app/benetnasch', 'migrate', '--allow-writes'
)

if ($VerifyIdempotency) {
    Write-Host 'Re-running migrations to verify applied checksums and idempotency...' -ForegroundColor Cyan
    Invoke-IntegrationCompose -Arguments @(
        'exec', '-T', 'backend',
        '/app/benetnasch', 'migrate', '--allow-writes'
    )
}

Write-Host 'Isolated database migrations applied.' -ForegroundColor Green
