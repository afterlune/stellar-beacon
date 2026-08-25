param(
    [switch]$NoBuild
)

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$blogDist = Join-Path $script:IntegrationRepoRoot 'web/blog/dist'
$adminDist = Join-Path $script:IntegrationRepoRoot 'web/admin/dist'
if (-not (Test-Path -LiteralPath $blogDist) -or -not (Test-Path -LiteralPath $adminDist)) {
    throw 'Frontend dist directories are missing. Run npm run build in web/blog and web/admin first.'
}

Invoke-IntegrationCompose -Arguments @('config', '--quiet')
$upArgs = @('up', '-d')
if (-not $NoBuild) {
    $upArgs += '--build'
}
Invoke-IntegrationCompose -Arguments $upArgs
Wait-IntegrationHttp -Uri 'http://127.0.0.1:18080/'
Write-Host 'Isolated integration stack is up.' -ForegroundColor Green
