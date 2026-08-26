$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$blogRoot = Join-Path $repoRoot 'web/blog'
$adminRoot = Join-Path $repoRoot 'web/admin'
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

Write-Host 'Building current backend binary for the isolated image...' -ForegroundColor Cyan
$backendBinary = Join-Path $repoRoot '.integration/backend/benetnasch'
$backendDirectory = Split-Path -Parent $backendBinary
New-Item -ItemType Directory -Path $backendDirectory -Force | Out-Null
$backendBuildEnvironment = @{}
foreach ($name in @('CGO_ENABLED', 'GOOS', 'GOARCH')) {
    $backendBuildEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
try {
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    Invoke-RequiredCommand -FilePath 'go' -ArgumentList @(
        'build',
        '-ldflags',
        '-w -s',
        '-trimpath',
        '-o',
        $backendBinary,
        '.'
    ) -WorkingDirectory $repoRoot
} finally {
    foreach ($name in $backendBuildEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $backendBuildEnvironment[$name], 'Process')
    }
}

Write-Host 'Updating isolated integration stack...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-up.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Repairing isolated database identity sequences...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-repair-sequences.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Seeding isolated integration data...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-seed.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Running isolated frontend/backend smoke test...' -ForegroundColor Cyan
Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
    '-NoLogo',
    '-NoProfile',
    '-ExecutionPolicy',
    'Bypass',
    '-File',
    (Join-Path $PSScriptRoot 'integration-smoke.ps1')
) -WorkingDirectory $repoRoot

Write-Host 'Isolated deployment is ready:' -ForegroundColor Green
Write-Host '  Blog:  http://127.0.0.1:18080'
Write-Host '  Admin: http://127.0.0.1:18008'
