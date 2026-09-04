param(
    [string]$AdminNextBaseUrl = '',
    [string]$ApiTarget = 'http://127.0.0.1:18008',
    [switch]$RequireAgentRoutes
)

$ErrorActionPreference = 'Stop'

function Assert-LoopbackUrl {
    param(
        [Parameter(Mandatory = $true)][string]$Value,
        [Parameter(Mandatory = $true)][int[]]$ExpectedPorts,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $uri = [Uri]$Value
    if ($uri.Scheme -ne 'http' -or $uri.Host -notin @('127.0.0.1', 'localhost') -or $uri.Port -notin $ExpectedPorts) {
        $ports = $ExpectedPorts -join ', '
        throw "$Name must point to an HTTP loopback URL on port(s) $ports; refusing to run against another target."
    }
    return $uri.AbsoluteUri.TrimEnd('/')
}

if ([string]::IsNullOrWhiteSpace($env:E2E_ADMIN_TOKEN)) {
    throw 'E2E_ADMIN_TOKEN must be set to an existing administrator token; this check never logs in.'
}

$safeApiTarget = Assert-LoopbackUrl -Value $ApiTarget -ExpectedPorts @(18008) -Name 'ApiTarget'
$safeAdminNextBaseUrl = ''
if (-not [string]::IsNullOrWhiteSpace($AdminNextBaseUrl)) {
    $safeAdminNextBaseUrl = Assert-LoopbackUrl -Value $AdminNextBaseUrl -ExpectedPorts @(18018) -Name 'AdminNextBaseUrl'
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$adminNextRoot = Join-Path $repoRoot 'web/admin-next'
$saved = @{}
foreach ($name in @('E2E_BASE_URL', 'E2E_REAL_INTEGRATION', 'E2E_REQUIRE_AGENT_ROUTES', 'VITE_ADMIN_API_TARGET', 'VITE_ADMIN_API_PRESERVE_API_PREFIX')) {
    $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

try {
    $env:E2E_REAL_INTEGRATION = '1'
    $env:E2E_REQUIRE_AGENT_ROUTES = if ($RequireAgentRoutes) { '1' } else { '0' }
    $env:VITE_ADMIN_API_TARGET = $safeApiTarget
    $env:VITE_ADMIN_API_PRESERVE_API_PREFIX = '1'
    if ($safeAdminNextBaseUrl) {
        $env:E2E_BASE_URL = $safeAdminNextBaseUrl
    } else {
        Remove-Item Env:E2E_BASE_URL -ErrorAction SilentlyContinue
    }

    Push-Location $adminNextRoot
    try {
        npm run test:e2e:integration -- --grep '@readonly' --reporter=line
        if ($LASTEXITCODE -ne 0) {
            throw "admin-next read-only browser integration failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
} finally {
    foreach ($name in $saved.Keys) {
        [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process')
    }
}

Write-Host 'admin-next read-only browser integration passed. No login, write, upload, migration, or container operation was performed.' -ForegroundColor Green
