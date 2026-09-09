$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$webRoot = Join-Path $repoRoot 'web'
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
Invoke-RequiredCommand -FilePath 'npm' -ArgumentList @('run', 'build:blog') -WorkingDirectory $webRoot

Write-Host 'Building admin frontend...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'npm' -ArgumentList @('run', 'build:admin') -WorkingDirectory $webRoot

Write-Host 'Updating isolated integration stack...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'up.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Repairing isolated database identity sequences...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'repair-sequences.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Seeding isolated integration data...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'seed.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Running isolated frontend/backend smoke test...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'smoke.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Isolated deployment is ready:' -ForegroundColor Green
Write-Host '  Blog:  http://127.0.0.1:18080'
Write-Host '  Admin: http://127.0.0.1:18008'
