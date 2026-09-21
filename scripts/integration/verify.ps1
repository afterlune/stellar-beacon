$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'common.ps1')
Import-IntegrationEnv

Write-Host 'Refreshing the isolated integration stack...' -ForegroundColor Cyan
& pwsh -NoLogo -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'deploy.ps1')
if ($LASTEXITCODE -ne 0) {
    throw "isolated integration deployment failed with exit code $LASTEXITCODE"
}

$blogBase = 'http://127.0.0.1:18080'
$adminBase = 'http://127.0.0.1:18008'

$loginBody = 'username=' + [uri]::EscapeDataString($env:E2E_ADMIN_EMAIL) + '&password=' + [uri]::EscapeDataString($env:E2E_ADMIN_PASSWORD)
$loginResponse = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/auth/login" -Method POST -Body $loginBody -ContentType 'application/x-www-form-urlencoded'
$loginPayload = Assert-IntegrationApiSuccess -Response $loginResponse -Name 'real integration admin login'
$adminToken = [string]$loginPayload.data.token
if ([string]::IsNullOrWhiteSpace($adminToken)) {
    throw 'real integration admin login did not return a token'
}

$env:BLOG_BASE_URL = $blogBase
$env:BLOG_VISUAL_BASE_URL = $blogBase
$env:E2E_BASE_URL = $adminBase
$env:E2E_REAL_INTEGRATION = '1'
$env:E2E_ADMIN_ALLOW_LOGIN = '1'
$env:E2E_ADMIN_TOKEN = $adminToken

Push-Location (Join-Path $script:IntegrationRepoRoot 'web')
try {
    Write-Host 'Running blog real backend integration tests...' -ForegroundColor Cyan
    & npm run test:e2e:integration --workspace=@stellar-beacon/blog
    if ($LASTEXITCODE -ne 0) { throw "blog integration tests failed with exit code $LASTEXITCODE" }

    Write-Host 'Running admin real backend integration tests...' -ForegroundColor Cyan
    & npm run test:e2e:integration --workspace=@stellar-beacon/admin-next
    if ($LASTEXITCODE -ne 0) { throw "admin integration tests failed with exit code $LASTEXITCODE" }

    Write-Host 'Running blog visual integration gate...' -ForegroundColor Cyan
    & npm run test:visual --workspace=@stellar-beacon/blog
    if ($LASTEXITCODE -ne 0) { throw "blog visual gate failed with exit code $LASTEXITCODE" }
} finally {
    Pop-Location
}

Write-Host 'Complete frontend/backend integration verification passed.' -ForegroundColor Green
