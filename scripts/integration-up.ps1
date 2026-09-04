param(
    [switch]$AllowContainerChanges,
    [switch]$NoBuild
)

$ErrorActionPreference = 'Stop'

if (-not $AllowContainerChanges) {
    throw 'Refusing to start or rebuild the isolated Compose project. Re-run with -AllowContainerChanges during an approved integration window.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$blogDist = Join-Path $script:IntegrationRepoRoot 'web/blog/dist'
$adminDist = Join-Path $script:IntegrationRepoRoot 'web/admin/dist'
$adminNextDist = Join-Path $script:IntegrationRepoRoot 'web/admin-next/dist'
if (-not (Test-Path -LiteralPath $blogDist) -or -not (Test-Path -LiteralPath $adminDist) -or -not (Test-Path -LiteralPath $adminNextDist)) {
    throw 'Frontend dist directories are missing. Run npm run build in web/blog, web/admin, and web/admin-next first.'
}

Invoke-IntegrationCompose -Arguments @('config', '--quiet')
$upArgs = @('up', '-d')
if (-not $NoBuild) {
    $upArgs += '--build'
}
Invoke-IntegrationCompose -Arguments $upArgs
Wait-IntegrationHttp -Uri 'http://127.0.0.1:18080/'
Assert-IntegrationMigrationStatusCommand
Write-Host 'Isolated integration stack is up.' -ForegroundColor Green
