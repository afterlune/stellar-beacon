param(
    [string]$BlogBaseUrl = 'http://127.0.0.1:18080',
    [string]$AdminNextBaseUrl = 'http://127.0.0.1:18018',
    [switch]$BlogOnly,
    [switch]$AdminNextOnly,
    [switch]$RequireAgentRoutes
)

$ErrorActionPreference = 'Stop'
if ($BlogOnly -and $AdminNextOnly) {
    throw 'BlogOnly and AdminNextOnly cannot be combined.'
}
$runBlog = -not $AdminNextOnly
$runAdminNext = -not $BlogOnly
if ($RequireAgentRoutes -and -not $runAdminNext) {
    throw 'RequireAgentRoutes requires the admin-next integration target.'
}
$repoRoot = Split-Path -Parent $PSScriptRoot
$blogRoot = Join-Path $repoRoot 'web/blog'
$adminNextRoot = Join-Path $repoRoot 'web/admin-next'
. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

function Assert-IsolatedUrl {
    param(
        [Parameter(Mandatory = $true)][string]$Value,
        [Parameter(Mandatory = $true)][int]$ExpectedPort,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $uri = [Uri]$Value
    if ($uri.Scheme -ne 'http' -or $uri.Host -notin @('127.0.0.1', 'localhost') -or $uri.Port -ne $ExpectedPort) {
        throw "$Name must point to the isolated loopback port $ExpectedPort; refusing to run browser integration against another target."
    }
    return $uri.AbsoluteUri.TrimEnd('/')
}

function Get-AdminToken {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl
    )

    $payload = @{
        username = $env:E2E_ADMIN_EMAIL
        password = $env:E2E_ADMIN_PASSWORD
    } | ConvertTo-Json -Compress

    try {
        $login = Invoke-RestMethod -Uri "$BaseUrl/api/users/login" -Method Post `
            -ContentType 'application/json' -Body $payload -TimeoutSec 15 -MaximumRedirection 0
    } catch {
        throw 'isolated admin login failed while preparing the read-only browser matrix.'
    }

    if ($null -eq $login -or $login.flag -ne $true -or [int]$login.code -ne 20000) {
        throw 'isolated admin login returned an unsuccessful result while preparing the read-only browser matrix.'
    }

    $token = ''
    if ($login.data -is [string]) {
        $token = [string]$login.data
    } elseif ($null -ne $login.data) {
        foreach ($field in @('token', 'accessToken', 'access_token')) {
            $property = $login.data.PSObject.Properties[$field]
            if ($null -ne $property -and -not [string]::IsNullOrWhiteSpace([string]$property.Value)) {
                $token = [string]$property.Value
                break
            }
        }
    }
    if ([string]::IsNullOrWhiteSpace($token)) {
        throw 'isolated admin login did not return a token for the read-only browser matrix.'
    }
    return $token
}

function Invoke-RequiredNpm {
    param(
        [Parameter(Mandatory = $true)][string]$WorkingDirectory,
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [switch]$AdminIntegration,
        [switch]$ReadOnly,
        [switch]$RequireAgentRoutes
    )

    $saved = @{}
    foreach ($name in @('E2E_BASE_URL', 'E2E_REAL_INTEGRATION', 'E2E_ADMIN_ALLOW_LOGIN', 'E2E_ADMIN_TOKEN', 'E2E_REQUIRE_AGENT_ROUTES')) {
        $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
    }
    try {
        $env:E2E_BASE_URL = $BaseUrl
        $env:E2E_REAL_INTEGRATION = '1'
        if ($AdminIntegration) {
            $env:E2E_ADMIN_ALLOW_LOGIN = '1'
            $env:E2E_ADMIN_TOKEN = Get-AdminToken -BaseUrl $BaseUrl
            $env:E2E_REQUIRE_AGENT_ROUTES = if ($RequireAgentRoutes) { '1' } else { '0' }
        }
        Push-Location $WorkingDirectory
        try {
            $playwrightArgs = @('--reporter=line')
            if ($ReadOnly) {
                $playwrightArgs += @('--grep', '@readonly')
            }
            & npm @(@('run', 'test:e2e:integration', '--') + $playwrightArgs)
            if ($LASTEXITCODE -ne 0) {
                throw "real integration browser test failed in $WorkingDirectory with exit code $LASTEXITCODE"
            }
        } finally {
            Pop-Location
        }
    } finally {
        foreach ($name in $saved.Keys) {
            [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process')
        }
    }
}

if ($runBlog) {
    $safeBlogBaseUrl = Assert-IsolatedUrl -Value $BlogBaseUrl -ExpectedPort 18080 -Name 'BlogBaseUrl'
    Write-Host "Running blog browser checks against $safeBlogBaseUrl ..." -ForegroundColor Cyan
    Invoke-RequiredNpm -WorkingDirectory $blogRoot -BaseUrl $safeBlogBaseUrl
}

if ($runAdminNext) {
    $safeAdminNextBaseUrl = Assert-IsolatedUrl -Value $AdminNextBaseUrl -ExpectedPort 18018 -Name 'AdminNextBaseUrl'
    if ([string]::IsNullOrWhiteSpace($env:E2E_ADMIN_EMAIL) -or [string]::IsNullOrWhiteSpace($env:E2E_ADMIN_PASSWORD)) {
        throw 'E2E_ADMIN_EMAIL and E2E_ADMIN_PASSWORD must be set before real admin browser integration.'
    }
    Write-Host "Running admin-next browser checks against $safeAdminNextBaseUrl ..." -ForegroundColor Cyan
    Invoke-RequiredNpm -WorkingDirectory $adminNextRoot -BaseUrl $safeAdminNextBaseUrl `
        -AdminIntegration -ReadOnly -RequireAgentRoutes:$RequireAgentRoutes
}

$target = if ($BlogOnly) { 'blog' } elseif ($AdminNextOnly) { 'admin-next' } else { 'blog and admin-next' }
Write-Host "Real isolated $target browser integration passed. No container or production Caddy operation was performed." -ForegroundColor Green
