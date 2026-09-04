[CmdletBinding()]
param(
    [ValidateRange(1, 64)]
    [int]$MinimumFreeHostGiB = 16,
    [ValidateRange(1024, 65536)]
    [int]$MinimumFreeGPUMiB = 5120,
    [ValidateRange(1, 64)]
    [int]$MinimumDockerMemoryGiB = 10,
    [ValidateRange(1, 64)]
    [int]$MinimumDockerFreeGiB = 12,
    [switch]$RequireRunningContainer
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

# Keep the optional service opt-in at the Compose layer as well as at the
# provider/configuration layer. This prevents a future launch command from
# silently making the memory-sensitive overlay part of a default project.
$qwenCompose = Join-Path $PSScriptRoot '..\docker-compose.integration.qwen.yaml'
if (-not (Test-Path -LiteralPath $qwenCompose -PathType Leaf)) {
    throw "Qwen Compose overlay is missing: $qwenCompose"
}
$qwenComposeText = Get-Content -Raw -LiteralPath $qwenCompose
if ($qwenComposeText -notmatch '(?m)^\s*profiles:\s*$' -or
        $qwenComposeText -notmatch '(?m)^\s*-\s*qwen-low-memory\s*$') {
    throw 'Qwen Compose overlay must remain behind the qwen-low-memory profile.'
}

function Convert-ToGiB {
    param(
        [Parameter(Mandatory = $true)]
        [long]$Bytes
    )

    return [math]::Round($Bytes / [double]1GB, 2)
}

if (-not [Environment]::OSVersion.Platform.ToString().Equals('Win32NT', [StringComparison]::OrdinalIgnoreCase)) {
    throw 'This preflight is intended for the Windows host that runs Docker Desktop/WSL.'
}

try {
    $operatingSystem = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop
} catch {
    throw 'Unable to read Windows physical-memory information; refusing to start the local Qwen service.'
}

$freeHostGiB = [math]::Round(([double]$operatingSystem.FreePhysicalMemory / 1MB), 2)
if ($freeHostGiB -lt $MinimumFreeHostGiB) {
    throw "Only $freeHostGiB GiB of host physical memory is available; require at least $MinimumFreeHostGiB GiB. Do not start Qwen."
}

if ($null -eq (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw 'docker is not available; refusing to start the local Qwen service.'
}

$LASTEXITCODE = 0
$dockerMemoryLine = & docker info --format '{{.MemTotal}}' 2>$null | Select-Object -First 1
$dockerMemoryText = if ($null -eq $dockerMemoryLine) { '' } else { ([string]$dockerMemoryLine).Trim() }
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($dockerMemoryText)) {
    throw 'Unable to read Docker memory capacity; refusing to start the local Qwen service.'
}

[long]$dockerMemoryBytes = 0
if (-not [long]::TryParse($dockerMemoryText, [Globalization.NumberStyles]::Integer, [Globalization.CultureInfo]::InvariantCulture, [ref]$dockerMemoryBytes)) {
    throw 'Docker returned an invalid memory capacity; refusing to start the local Qwen service.'
}

$dockerMemoryGiB = Convert-ToGiB -Bytes $dockerMemoryBytes
if ($dockerMemoryGiB -lt $MinimumDockerMemoryGiB) {
    throw "Docker reports $dockerMemoryGiB GiB available; require at least $MinimumDockerMemoryGiB GiB. Do not start Qwen."
}

if ($null -eq (Get-Command nvidia-smi -ErrorAction SilentlyContinue)) {
    throw 'nvidia-smi is not available; refusing to start the GPU-backed local Qwen service.'
}

$gpuLines = @(& nvidia-smi --query-gpu=index,memory.free --format=csv,noheader,nounits 2>$null)
if ($LASTEXITCODE -ne 0 -or $gpuLines.Count -eq 0) {
    throw 'Unable to read GPU free-memory information; refusing to start the local Qwen service.'
}

[int64]$minFreeGPUMiB = [int64]::MaxValue
foreach ($gpuLine in $gpuLines) {
    $gpuParts = ([string]$gpuLine).Split(',', 2)
    if ($gpuParts.Count -ne 2) {
        throw "nvidia-smi returned an invalid GPU memory value: $gpuLine"
    }
    [int64]$freeGPUMiB = 0
    if (-not [int64]::TryParse($gpuParts[1].Trim(), [Globalization.NumberStyles]::Integer, [Globalization.CultureInfo]::InvariantCulture, [ref]$freeGPUMiB) -or $freeGPUMiB -lt 0) {
        throw "nvidia-smi returned an invalid GPU free-memory value: $gpuLine"
    }
    if ($freeGPUMiB -lt $minFreeGPUMiB) {
        $minFreeGPUMiB = $freeGPUMiB
    }
}
if ($minFreeGPUMiB -lt $MinimumFreeGPUMiB) {
    throw "A visible GPU has only $minFreeGPUMiB MiB free; require at least $MinimumFreeGPUMiB MiB on every visible GPU. Do not start Qwen."
}

function Convert-DockerMemoryTextToBytes {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Text
    )

    if ($Text -notmatch '^\s*(?<amount>[0-9]+(?:\.[0-9]+)?)\s*(?<unit>B|KiB|MiB|GiB|TiB)\s*$') {
        throw "Docker returned an invalid container memory value: $Text"
    }
    [double]$amount = $Matches.amount
    $multiplier = switch ($Matches.unit) {
        'B'   { 1.0 }
        'KiB' { 1KB }
        'MiB' { 1MB }
        'GiB' { 1GB }
        'TiB' { 1TB }
    }
    return [int64][math]::Round($amount * $multiplier)
}

$dockerStatsLines = @(& docker stats --no-stream --format '{{.MemUsage}}' 2>$null)
if ($LASTEXITCODE -ne 0) {
    throw 'Unable to read current Docker container memory usage; refusing to start the local Qwen service.'
}
[int64]$dockerUsedBytes = 0
foreach ($dockerStatsLine in $dockerStatsLines) {
    $usedText = ([string]$dockerStatsLine -split '/', 2)[0].Trim()
    [int64]$dockerUsedBytes += Convert-DockerMemoryTextToBytes -Text $usedText
}
[int64]$zeroBytes = 0
$dockerFreeBytes = [math]::Max($zeroBytes, $dockerMemoryBytes - $dockerUsedBytes)
$dockerFreeGiB = Convert-ToGiB -Bytes $dockerFreeBytes
if ($dockerFreeGiB -lt $MinimumDockerFreeGiB) {
    throw "Docker has only $dockerFreeGiB GiB free after current containers; require at least $MinimumDockerFreeGiB GiB before starting Qwen."
}

if ($RequireRunningContainer) {
    $containerIDs = @(
        & docker ps -aq `
            --filter 'label=com.docker.compose.project=benetnasch-integration' `
            --filter 'label=com.docker.compose.service=qwen-embedding' 2>$null |
            Where-Object { $_ -match '^[0-9a-f]+$' }
    )
    if ($LASTEXITCODE -ne 0 -or $containerIDs.Count -ne 1) {
        throw 'Exactly one isolated qwen-embedding container must exist before local smoke; recreate it with the qwen-low-memory profile.'
    }

    $containerJSON = (& docker inspect --format '{{json .}}' $containerIDs[0] 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($containerJSON)) {
        throw 'Unable to inspect the isolated qwen-embedding container; refusing local smoke.'
    }
    try {
        $container = $containerJSON | ConvertFrom-Json
    } catch {
        throw 'The isolated qwen-embedding container returned malformed inspection data; refusing local smoke.'
    }
    if ([string]$container.State.Status -ne 'running') {
        throw 'The isolated qwen-embedding container is not running; refusing local smoke.'
    }

    [int64]$containerMemoryBytes = $container.HostConfig.Memory
    [int64]$containerSwapBytes = $container.HostConfig.MemorySwap
    [int64]$maxContainerMemoryBytes = 4GB
    if ($containerMemoryBytes -le 0 -or $containerMemoryBytes -gt $maxContainerMemoryBytes) {
        throw 'The running qwen-embedding container has no valid 4 GiB memory cap; recreate it with docker-compose.integration.qwen.yaml and --profile qwen-low-memory.'
    }
    if ($containerSwapBytes -ne $containerMemoryBytes) {
        throw 'The running qwen-embedding container allows swap beyond its memory cap; recreate it with the low-memory overlay.'
    }
}

Write-Host ("Qwen memory preflight passed: host free {0:N2} GiB, minimum GPU free {1:N0} MiB, Docker capacity {2:N2} GiB, Docker free after current containers {3:N2} GiB. The optional container remains capped at 4 GiB and must still be used for one-request smoke only." -f $freeHostGiB, $minFreeGPUMiB, $dockerMemoryGiB, $dockerFreeGiB) -ForegroundColor Green
