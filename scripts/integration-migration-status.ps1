param()

$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv
Assert-IntegrationMigrationStatusCommand

# This command is deliberately read-only. It does not start, recreate, or
# modify any Compose service and it never invokes the migration Up command.
Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'backend',
    '/app/benetnasch', 'migrate', 'status'
)

Write-Host 'Isolated migration status inspected without applying migrations.' -ForegroundColor Green
