[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ModelPath,
    [switch]$Start,
    [switch]$IncludeBackend,
    [switch]$AllowContainerChanges,
    [ValidateRange(1, 64)]
    [int]$MinimumFreeHostGiB = 16,
    [ValidateRange(1024, 65536)]
    [int]$MinimumFreeGPUMiB = 5120,
    [ValidateRange(1, 64)]
    [int]$MinimumDockerMemoryGiB = 10,
    [ValidateRange(1, 64)]
    [int]$MinimumDockerFreeGiB = 12
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($Start -and -not $AllowContainerChanges) {
    throw 'Refusing to start the isolated Qwen service without -AllowContainerChanges.'
}
if ($IncludeBackend -and -not $Start) {
    throw '-IncludeBackend requires -Start; preflight mode never changes a container.'
}
if ($IncludeBackend -and -not $AllowContainerChanges) {
    throw 'Refusing to recreate the isolated backend without -AllowContainerChanges.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

try {
    $modelItem = Get-Item -LiteralPath $ModelPath -Force -ErrorAction Stop
} catch {
    throw 'Qwen model directory does not exist; no container action was attempted.'
}
if (-not $modelItem.PSIsContainer) {
    throw 'ModelPath must point to a directory; no container action was attempted.'
}
if (($modelItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
    throw 'ModelPath must not be a symlink or reparse point; no container action was attempted.'
}
$modelFullPath = $modelItem.FullName
$repoFullPath = (Get-Item -LiteralPath $script:IntegrationRepoRoot -Force).FullName
$repoPrefix = $repoFullPath.TrimEnd('\') + '\'
if ($modelFullPath.Equals($repoFullPath, [StringComparison]::OrdinalIgnoreCase) -or
    $modelFullPath.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'ModelPath must remain outside the repository; no container action was attempted.'
}
$env:QWEN_MODEL_PATH = $modelFullPath

$memoryCheck = Join-Path $PSScriptRoot 'check-qwen-memory.ps1'
& $memoryCheck `
    -MinimumFreeHostGiB $MinimumFreeHostGiB `
    -MinimumFreeGPUMiB $MinimumFreeGPUMiB `
    -MinimumDockerMemoryGiB $MinimumDockerMemoryGiB `
    -MinimumDockerFreeGiB $MinimumDockerFreeGiB
if ($LASTEXITCODE -ne 0) {
    throw 'Qwen memory preflight failed; no container action was attempted.'
}

$overlay = Join-Path $script:IntegrationRepoRoot 'docker-compose.integration.qwen.yaml'
$composeArgs = @(
    'compose',
    '-p', $script:IntegrationProject,
    '--env-file', $script:IntegrationEnvFile,
    '--profile', 'qwen-low-memory',
    '-f', $script:IntegrationComposeFile,
    '-f', $overlay
)
& docker @($composeArgs + @('config', '--quiet'))
if ($LASTEXITCODE -ne 0) {
    throw 'Qwen Compose configuration validation failed; no container action was attempted.'
}

if (-not $Start) {
    Write-Host 'Qwen low-memory preflight passed. No container was started or recreated.' -ForegroundColor Green
    exit 0
}

# Re-check immediately before the write operation because the host may have
# lost memory while Compose configuration was being validated.
& $memoryCheck `
    -MinimumFreeHostGiB $MinimumFreeHostGiB `
    -MinimumFreeGPUMiB $MinimumFreeGPUMiB `
    -MinimumDockerMemoryGiB $MinimumDockerMemoryGiB `
    -MinimumDockerFreeGiB $MinimumDockerFreeGiB
if ($LASTEXITCODE -ne 0) {
    throw 'Qwen memory preflight failed immediately before start; no container action was attempted.'
}

$services = @('qwen-embedding')
if ($IncludeBackend) {
    $services += 'backend'
}
& docker @($composeArgs + @('up', '-d', '--no-deps', '--force-recreate') + $services)
if ($LASTEXITCODE -ne 0) {
    throw 'Isolated Qwen Compose start failed.'
}

Write-Host ("Started only the explicitly requested isolated service(s): {0}. Article indexing remains disabled by this overlay." -f ($services -join ', ')) -ForegroundColor Green
