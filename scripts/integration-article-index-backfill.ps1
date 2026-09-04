param(
    [ValidateSet('start', 'run', 'resume', 'status')]
    [string]$Action = 'start',
    [string]$IndexUid = 'article_chunks_v2',
    [string]$RunId = 'article-index-backfill-v2',
    [int]$PageSize = 1,
    [int]$MaxArticlesPerRun = 1,
    [int]$RunTimeoutSeconds = 600,
    [switch]$AllowExternalProviderCalls,
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'

$needsExternalProvider = $Action -in @('start', 'run')
$needsWrites = $Action -in @('start', 'run', 'resume')
if (-not $AllowExternalProviderCalls -and $needsExternalProvider) {
    throw 'Refusing isolated article-index backfill. start/run require -AllowExternalProviderCalls.'
}
if (-not $AllowWrites -and $needsWrites) {
    throw 'Refusing isolated article-index backfill. start/run require -AllowExternalProviderCalls -AllowWrites; resume requires -AllowWrites.'
}

if ($PageSize -lt 1 -or $PageSize -gt 25) {
    throw 'PageSize must be between 1 and 25 for the low-memory backfill runner.'
}
if ($MaxArticlesPerRun -lt 0) {
    throw 'MaxArticlesPerRun cannot be negative; use 0 explicitly only for an approved unbounded run.'
}
if ($RunTimeoutSeconds -le 0) {
    throw 'RunTimeoutSeconds must be positive.'
}
if ($IndexUid -notmatch '^article_chunks_[a-z0-9][a-z0-9_-]{0,30}$') {
    throw 'IndexUid must be a versioned article_chunks UID.'
}
if ($RunId -notmatch '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$') {
    throw 'RunId contains unsupported characters or exceeds 128 bytes.'
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$sourceConfig = Join-Path $repoRoot 'resource/config-integration.yaml'
if (-not (Test-Path -LiteralPath $sourceConfig -PathType Leaf)) {
    throw "Integration configuration is missing: $sourceConfig"
}

# The Go backfill command derives its target index from the configuration
# loaded by the running backend; this wrapper does not pass IndexUid through to
# the command. Fail closed instead of checking one UID and silently writing
# another when an operator supplies an unsupported candidate.
$configText = Get-Content -Raw -LiteralPath $sourceConfig
$indexVersionMatch = [regex]::Match($configText, '(?m)^\s+index_version:\s*"([^"]+)"\s*$')
if (-not $indexVersionMatch.Success) {
    throw "Unable to determine the configured article embedding index version from $sourceConfig."
}
$configuredIndexUid = 'article_chunks_' + $indexVersionMatch.Groups[1].Value.Trim().ToLowerInvariant()
if ($IndexUid -ne $configuredIndexUid) {
    throw "IndexUid '$IndexUid' does not match the running integration configuration target '$configuredIndexUid'; this wrapper cannot target an alternate UID."
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$composeArgs = @(Get-IntegrationComposeArgs)
$backendRouteCheck = $composeArgs + @(
    'exec', '-T', 'backend', '/bin/sh', '-c',
    'case "$BENETNASCH_AI_LOCAL_EMBEDDING" in 1|t|T|TRUE|True|true) exit 42;; esac; test -n "$ALIBAILIAN_API_KEY" && test "$BENETNASCH_AI_ENABLED" = "true" && test "$BENETNASCH_AI_ARTICLE_INDEXING" = "true"'
)
if ($needsExternalProvider) {
    & docker @backendRouteCheck 2>$null
    if ($LASTEXITCODE -eq 42) {
        throw 'The local Qwen/SGLang route is smoke-only under the low-memory profile; article-index backfill is intentionally refused. Use a separately approved, resource-tested indexing profile.'
    }
    if ($LASTEXITCODE -ne 0) {
        throw 'running isolated backend is missing the AliBailian embedding key or enabled article-indexing flags; local Qwen smoke routing cannot be used for backfill.'
    }
}

function Assert-EnabledFlag {
    param([Parameter(Mandatory = $true)][string]$Name)

    $parsed = $false
    if (-not [bool]::TryParse([string](Get-Item "Env:$Name" -ErrorAction SilentlyContinue).Value, [ref]$parsed) -or -not $parsed) {
        throw "$Name must be explicitly set to true in .env.integration and present in the running backend; the script never changes rollout flags."
    }
}

if ($needsExternalProvider) {
    Assert-EnabledFlag -Name 'BENETNASCH_AI_ENABLED'
    Assert-EnabledFlag -Name 'BENETNASCH_AI_ARTICLE_INDEXING'
}

if ($needsExternalProvider -and [string]::IsNullOrWhiteSpace($env:ALIBAILIAN_API_KEY)) {
    throw 'ALIBAILIAN_API_KEY must be set in .env.integration for the configured embedding route; the value is never printed.'
}
if ($needsExternalProvider -and [string]::IsNullOrWhiteSpace($env:MEILI_MASTER_KEY)) {
    throw 'MEILI_MASTER_KEY must be set in .env.integration; the value is never printed.'
}

if ($needsExternalProvider) {
    $headers = @{ Authorization = "Bearer $env:MEILI_MASTER_KEY" }
    try {
        $index = Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:17700/indexes/$IndexUid" -Headers $headers -TimeoutSec 10
    } catch {
        throw "isolated candidate index '$IndexUid' is not available; provision it explicitly before backfill."
    }
    if ([string]$index.uid -ne $IndexUid -or [string]$index.primaryKey -ne 'id') {
        throw "isolated candidate index '$IndexUid' does not match the required id primary-key contract."
    }
}

Write-Host "Running isolated article-index backfill action '$Action' for $IndexUid..." -ForegroundColor Cyan
$commandArgs = $composeArgs + @(
    'exec', '-T', 'backend', '/app/benetnasch',
    'article-index', 'backfill', $Action,
    '--run-id', $RunId
)
if ($Action -in @('start', 'run')) {
    $commandArgs += @(
        '--page-size', $PageSize,
        '--max-articles-per-run', $MaxArticlesPerRun,
        '--run-timeout', ("{0}s" -f $RunTimeoutSeconds),
        '--allow-writes'
    )
} elseif ($Action -eq 'resume') {
    $commandArgs += '--allow-writes'
}
& docker @commandArgs
if ($LASTEXITCODE -ne 0) {
    throw "isolated article-index backfill action '$Action' failed with exit code $LASTEXITCODE"
}

Write-Host 'Isolated article-index backfill command completed. Run the read-only quality/P95 gate before any swap.' -ForegroundColor Green
