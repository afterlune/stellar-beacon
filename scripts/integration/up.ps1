param(
    [switch]$NoBuild
)

. (Join-Path $PSScriptRoot 'common.ps1')
Import-IntegrationEnv

$blogDist = Join-Path $script:IntegrationRepoRoot 'web/apps/blog/dist'
$adminDist = Join-Path $script:IntegrationRepoRoot 'web/apps/admin-next/dist'
if (-not (Test-Path -LiteralPath $blogDist) -or -not (Test-Path -LiteralPath $adminDist)) {
    throw 'Frontend dist directories are missing. Run npm run build:blog and npm run build:admin from web first.'
}

Invoke-IntegrationCompose -Arguments @('config', '--quiet')
$upArgs = @('up', '-d')
if (-not $NoBuild) {
    $upArgs += '--build'
}
Invoke-IntegrationCompose -Arguments $upArgs
Wait-IntegrationHttp -Uri 'http://127.0.0.1:18080/'

Write-Host 'Applying versioned database migrations...' -ForegroundColor Cyan
Invoke-IntegrationCompose -Arguments @('run', '--rm', '--no-deps', 'backend', '/app/stellar-beacon', 'migrate')
Write-Host 'Isolated integration stack is up.' -ForegroundColor Green
