param(
    [switch]$AllowContainerChanges,
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'

if (-not $AllowContainerChanges -or -not $AllowWrites) {
    throw 'Refusing the isolated deployment because it changes containers and writes test data. Re-run with -AllowContainerChanges -AllowWrites during an approved integration window.'
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$blogRoot = Join-Path $repoRoot 'web/blog'
$adminRoot = Join-Path $repoRoot 'web/admin'
$adminNextRoot = Join-Path $repoRoot 'web/admin-next'
$envFile = Join-Path $repoRoot '.env.integration'

if (-not (Test-Path -LiteralPath $envFile)) {
    throw "Missing $envFile. Copy .env.integration.example to .env.integration first."
}

function Invoke-RequiredCommand {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string[]]$ArgumentList,
        [Parameter(Mandatory = $true)][string]$WorkingDirectory
    )

    Push-Location $WorkingDirectory
    try {
        & $FilePath @ArgumentList
        if ($LASTEXITCODE -ne 0) {
            throw "$FilePath $($ArgumentList -join ' ') failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
}

Write-Host 'Building blog frontend...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'npm' -ArgumentList @('run', 'build') -WorkingDirectory $blogRoot

Write-Host 'Building admin frontend...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'npm' -ArgumentList @('run', 'build') -WorkingDirectory $adminRoot

Write-Host 'Building admin-next preview frontend...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'npm' -ArgumentList @('run', 'build') -WorkingDirectory $adminNextRoot

Write-Host 'Updating isolated integration stack...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-up.ps1'),
    '-AllowContainerChanges'
) -WorkingDirectory $repoRoot

Write-Host 'Applying explicit migrations to the isolated database...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-migrate.ps1'),
    '-AllowWrites',
    '-VerifyIdempotency'
) -WorkingDirectory $repoRoot

Write-Host 'Repairing isolated database identity sequences...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-repair-sequences.ps1'),
    '-AllowWrites'
) -WorkingDirectory $repoRoot

Write-Host 'Seeding isolated integration data...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-seed.ps1'),
    '-AllowWrites'
) -WorkingDirectory $repoRoot

Write-Host 'Running isolated frontend/backend smoke test...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-smoke.ps1'),
    '-AllowWrites'
) -WorkingDirectory $repoRoot

Write-Host 'Isolated deployment is ready:' -ForegroundColor Green
Write-Host '  Blog:  http://127.0.0.1:18080'
Write-Host '  Admin: http://127.0.0.1:18008'
Write-Host '  Admin-next preview: http://127.0.0.1:18018'
