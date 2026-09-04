[CmdletBinding()]
param(
    [string]$OutputPath,
    [ValidateRange(1, 32)]
    [int]$Parallelism = 1,
    [switch]$Force
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repoRoot = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($OutputPath)) {
    $OutputPath = Join-Path ([System.IO.Path]::GetTempPath()) (
        'benetnasch-linux-amd64-{0}' -f ([guid]::NewGuid().ToString('N'))
    )
}

$resolvedOutput = [System.IO.Path]::GetFullPath($OutputPath)
$outputDirectory = Split-Path -Parent $resolvedOutput
if (-not (Test-Path -LiteralPath $outputDirectory -PathType Container)) {
    New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null
}
if ((Test-Path -LiteralPath $resolvedOutput) -and -not $Force) {
    throw "Refusing to overwrite existing build output '$resolvedOutput'. Pass -Force explicitly."
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go was not found on PATH. Install the Windows Go toolchain before using this script.'
}

$goHost = @(& go env GOHOSTOS GOHOSTARCH)
if ($LASTEXITCODE -ne 0 -or $goHost.Count -lt 2) {
    throw 'Unable to inspect the Go host toolchain. Run this script with a working native Windows Go installation.'
}
$goHostOS = ([string]$goHost[0]).Trim()
$goHostArch = ([string]$goHost[1]).Trim()
if ($goHostOS -ne 'windows') {
    throw "This helper must use a native Windows Go toolchain; detected ${goHostOS}/${goHostArch}. Do not invoke it from WSL."
}

$previousGoOS = $env:GOOS
$previousGoArch = $env:GOARCH
$previousCGO = $env:CGO_ENABLED
$previousGoMaxProcs = $env:GOMAXPROCS

try {
    # These variables are scoped to this PowerShell process and make the build
    # independent of the host OS and any parent-shell Go configuration.
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    $env:GOMAXPROCS = [string]$Parallelism

    Push-Location $repoRoot
    try {
        $version = (& go version | Out-String).Trim()
        Write-Host "Building Linux/amd64 backend with the native Windows Go toolchain ($version)..." -ForegroundColor Cyan
        & go build -p $Parallelism -trimpath -ldflags '-w -s' -o $resolvedOutput .
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }

    $bytes = [System.IO.File]::ReadAllBytes($resolvedOutput)
    if ($bytes.Length -lt 4 -or $bytes[0] -ne 0x7f -or $bytes[1] -ne 0x45 -or
        $bytes[2] -ne 0x4c -or $bytes[3] -ne 0x46) {
        throw "Build output '$resolvedOutput' is not an ELF binary. Check GOOS/GOARCH and the host toolchain."
    }

    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $resolvedOutput).Hash
    Write-Host "Linux/amd64 ELF written to: $resolvedOutput" -ForegroundColor Green
    Write-Host "SHA-256: $hash"
    Write-Host 'No Docker, WSL, Compose, database, search index, object storage, or running container was accessed.' -ForegroundColor Yellow
} finally {
    if ($null -eq $previousGoOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $previousGoOS }
    if ($null -eq $previousGoArch) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $previousGoArch }
    if ($null -eq $previousCGO) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue } else { $env:CGO_ENABLED = $previousCGO }
    if ($null -eq $previousGoMaxProcs) { Remove-Item Env:GOMAXPROCS -ErrorAction SilentlyContinue } else { $env:GOMAXPROCS = $previousGoMaxProcs }
}
