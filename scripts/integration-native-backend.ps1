[CmdletBinding()]
param(
    [switch]$AllowContainerChanges,
    [string]$BinaryPath,
    [switch]$Build
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if (-not $AllowContainerChanges) {
    throw 'Refusing to replace a container binary. Re-run with -AllowContainerChanges during an approved isolated integration window.'
}
if (($Build -and -not [string]::IsNullOrWhiteSpace($BinaryPath)) -or
    (-not $Build -and [string]::IsNullOrWhiteSpace($BinaryPath))) {
    throw 'Choose exactly one source: -Build or -BinaryPath <native Linux/amd64 ELF>.'
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$sourceConfig = Join-Path $repoRoot 'resource/config-integration.yaml'

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
            throw "$FilePath failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
}

if (-not (Test-Path -LiteralPath $sourceConfig -PathType Leaf)) {
    throw "Integration configuration is missing: $sourceConfig"
}
$configItem = Get-Item -LiteralPath $sourceConfig -Force
if (($configItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
    throw 'Integration configuration must not be a symlink or reparse point.'
}

if ($Build) {
    $nativeBinary = Join-Path ([IO.Path]::GetTempPath()) (
        'benetnasch-integration-native-{0}' -f ([guid]::NewGuid().ToString('N'))
    )
    Invoke-RequiredCommand -FilePath 'pwsh' -ArgumentList @(
        '-NoLogo',
        '-NoProfile',
        '-ExecutionPolicy',
        'Bypass',
        '-File',
        (Join-Path $PSScriptRoot 'build-linux-amd64.ps1'),
        '-OutputPath',
        $nativeBinary,
        '-Parallelism',
        '1'
    ) -WorkingDirectory $repoRoot
} else {
    $nativeBinary = [IO.Path]::GetFullPath($BinaryPath)
}

if (-not (Test-Path -LiteralPath $nativeBinary -PathType Leaf)) {
    throw "Native backend binary is missing: $nativeBinary"
}
$resolvedBinaryPath = [IO.Path]::GetFullPath((Resolve-Path -LiteralPath $nativeBinary).Path)
$resolvedRepoRoot = [IO.Path]::GetFullPath($repoRoot)
if ($resolvedBinaryPath.Equals($resolvedRepoRoot, [StringComparison]::OrdinalIgnoreCase) -or
    $resolvedBinaryPath.StartsWith($resolvedRepoRoot + '\', [StringComparison]::OrdinalIgnoreCase) -or
    $resolvedBinaryPath.StartsWith($resolvedRepoRoot + '/', [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Native backend binary must be outside the repository; keep build artifacts out of Git.'
}
if ($resolvedBinaryPath -match '(?i)(^|[\\/])\.codebuddy([\\/]|$)') {
    throw 'Native backend binary must not come from the .codebuddy evaluation directory.'
}
$binaryItem = Get-Item -LiteralPath $nativeBinary -Force
if (($binaryItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
    throw 'Native backend binary must not be a symlink or reparse point.'
}
$binaryBytes = [IO.File]::ReadAllBytes($nativeBinary)
if ($binaryBytes.Length -lt 4 -or $binaryBytes[0] -ne 0x7f -or $binaryBytes[1] -ne 0x45 -or
    $binaryBytes[2] -ne 0x4c -or $binaryBytes[3] -ne 0x46) {
    throw 'Native backend binary is not an ELF file.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

function Invoke-DockerChecked {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)

    & docker @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "docker command failed with exit code $LASTEXITCODE"
    }
}

$composeArgs = @(Get-IntegrationComposeArgs)
$backendOutput = & docker @composeArgs ps -q backend
if ($LASTEXITCODE -ne 0) {
    throw 'Unable to inspect the isolated backend through Compose.'
}
$backendIDs = @($backendOutput | ForEach-Object { ([string]$_).Trim() } | Where-Object { $_ -ne '' })
if ($backendIDs.Count -ne 1) {
    throw "Expected exactly one isolated backend container, found $($backendIDs.Count)."
}
$backendID = $backendIDs[0]

$projectLabel = (& docker inspect --format '{{ index .Config.Labels "com.docker.compose.project" }}' $backendID 2>$null | Out-String).Trim()
$serviceLabel = (& docker inspect --format '{{ index .Config.Labels "com.docker.compose.service" }}' $backendID 2>$null | Out-String).Trim()
if ($projectLabel -ne 'benetnasch-integration' -or $serviceLabel -ne 'backend') {
    throw 'Refusing a container that is not the benetnasch-integration backend service.'
}
$state = (& docker inspect --format '{{.State.Status}}' $backendID 2>$null | Out-String).Trim()
if ($state -ne 'running') {
    throw "Isolated backend must already be running; current state is '$state'."
}

$backupRoot = Join-Path ([IO.Path]::GetTempPath()) (
    'benetnasch-integration-native-backup-{0}' -f ([guid]::NewGuid().ToString('N'))
)
New-Item -ItemType Directory -Path $backupRoot -Force | Out-Null
$backupBinary = Join-Path $backupRoot 'benetnasch'
$backupConfig = Join-Path $backupRoot 'config-integration.yaml'
Invoke-DockerChecked -Arguments @('cp', "${backendID}:/app/benetnasch", $backupBinary)
Invoke-DockerChecked -Arguments @('cp', "${backendID}:/app/resource/config-integration.yaml", $backupConfig)

$hostHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $nativeBinary).Hash.ToLowerInvariant()
Invoke-DockerChecked -Arguments @('cp', $nativeBinary, "${backendID}:/app/benetnasch")
Invoke-DockerChecked -Arguments @('cp', $sourceConfig, "${backendID}:/app/resource/config-integration.yaml")
Invoke-DockerChecked -Arguments @('exec', $backendID, 'chmod', '0755', '/app/benetnasch')
Invoke-DockerChecked -Arguments @('restart', $backendID)

$state = (& docker inspect --format '{{.State.Status}}' $backendID 2>$null | Out-String).Trim()
if ($state -ne 'running') {
    throw "Isolated backend did not return to running state; current state is '$state'. Backup: $backupRoot"
}
$containerHashOutput = (& docker exec $backendID sha256sum /app/benetnasch 2>$null | Out-String).Trim()
if ($containerHashOutput -notmatch '^([0-9a-fA-F]{64})\s') {
    throw "Unable to verify the isolated backend binary hash. Backup: $backupRoot"
}
$containerHash = $Matches[1].ToLowerInvariant()
if ($containerHash -ne $hostHash) {
    throw "Native backend hash mismatch after copy. Backup: $backupRoot"
}

Write-Host 'Isolated backend now uses the native Linux/amd64 binary.' -ForegroundColor Green
Write-Host "Binary SHA-256: $hostHash"
Write-Host "Rollback backup: $backupRoot"
Write-Host 'Only the benetnasch-integration backend container was restarted; no database, search index, object storage, Caddy, or production target was operated.' -ForegroundColor Yellow
