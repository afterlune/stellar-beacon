param(
    [string]$AdminNextBaseUrl = 'http://127.0.0.1:18018',
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'
if (-not $AllowWrites) {
    throw 'Refusing to run admin-next CRUD integration because it writes isolated database records. Re-run with -AllowWrites during an approved isolated integration window.'
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$adminNextRoot = Join-Path $repoRoot 'web/admin-next'
. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$uri = [Uri]$AdminNextBaseUrl
if ($uri.Scheme -ne 'http' -or $uri.Host -notin @('127.0.0.1', 'localhost') -or $uri.Port -ne 18018) {
    throw 'AdminNextBaseUrl must point to isolated loopback port 18018; refusing to run CRUD integration against another target.'
}
if ([string]::IsNullOrWhiteSpace($env:E2E_ADMIN_EMAIL) -or [string]::IsNullOrWhiteSpace($env:E2E_ADMIN_PASSWORD)) {
    throw 'E2E_ADMIN_EMAIL and E2E_ADMIN_PASSWORD must be set in .env.integration before running admin-next CRUD integration.'
}

$saved = @{}
foreach ($name in @('E2E_BASE_URL', 'E2E_REAL_INTEGRATION', 'E2E_ADMIN_ALLOW_LOGIN')) {
    $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
try {
    $env:E2E_BASE_URL = $uri.AbsoluteUri.TrimEnd('/')
    $env:E2E_REAL_INTEGRATION = '1'
    $env:E2E_ADMIN_ALLOW_LOGIN = '1'
    Push-Location $adminNextRoot
    try {
        npm run test:e2e:integration -- --grep '@crud' --reporter=line
        if ($LASTEXITCODE -ne 0) {
            throw "admin-next CRUD browser integration failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
} finally {
    foreach ($name in $saved.Keys) {
        [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process')
    }
}

Write-Host 'admin-next isolated CRUD browser integration passed. Only the authorized isolated database was written.' -ForegroundColor Green
