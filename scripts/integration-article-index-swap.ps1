[CmdletBinding()]
param(
    [string]$CurrentIndex = 'article_chunks_v1',
    [string]$CurrentIndexVersion = 'v1',
    [string]$CurrentProvider,
    [string]$CurrentModel,
    [string]$CurrentModelVersion,
    [int]$CurrentDimension = 1024,
    [int]$CurrentBatchSize = 32,
    [string]$CandidateIndex = 'article_chunks_v2',
    [string]$QualityEvidencePath,
    [int]$TargetMilliseconds = 300,
    [int64]$MinimumDocumentCount = 1,
    [int64]$CurrentMinimumDocumentCount = 1,
    [ValidateRange(1, 8760)]
    [int]$MaximumEvidenceAgeHours = 24,
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if (-not $AllowWrites) {
    throw 'Refusing isolated article-index swap. Re-run with -AllowWrites during an approved integration window.'
}

if ($CurrentIndex -notmatch '^article_chunks_[a-z0-9][a-z0-9_-]{0,30}$' -or
    $CandidateIndex -notmatch '^article_chunks_[a-z0-9][a-z0-9_-]{0,30}$') {
    throw 'CurrentIndex and CandidateIndex must be versioned article_chunks UIDs.'
}
if ($CurrentIndex -eq $CandidateIndex) {
    throw 'CurrentIndex and CandidateIndex must differ.'
}
if ($CurrentIndexVersion -notmatch '^[a-z0-9][a-z0-9_-]{0,30}$') {
    throw 'CurrentIndexVersion is invalid.'
}
if ([string]::IsNullOrWhiteSpace($CurrentProvider) -or
    [string]::IsNullOrWhiteSpace($CurrentModel) -or
    [string]::IsNullOrWhiteSpace($CurrentModelVersion)) {
    throw 'The current embedding contract is required.'
}
if ($CurrentDimension -le 0 -or $CurrentBatchSize -le 0 -or $CurrentBatchSize -gt 1024) {
    throw 'CurrentDimension must be positive and CurrentBatchSize must be between 1 and 1024.'
}
if ($TargetMilliseconds -le 0 -or $MinimumDocumentCount -le 0 -or $CurrentMinimumDocumentCount -le 0) {
    throw 'TargetMilliseconds, MinimumDocumentCount, and CurrentMinimumDocumentCount must be positive.'
}
if ([string]::IsNullOrWhiteSpace($QualityEvidencePath)) {
    throw 'QualityEvidencePath is required; an empty or unverified candidate must never be swapped.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

function Assert-EnabledFlag {
    param([Parameter(Mandatory = $true)][string]$Name)

    $parsed = $false
    if (-not [bool]::TryParse([string](Get-Item "Env:$Name" -ErrorAction SilentlyContinue).Value, [ref]$parsed) -or -not $parsed) {
        throw "$Name must be explicitly set to true in .env.integration; the script never changes rollout flags."
    }
}

Assert-EnabledFlag -Name 'BENETNASCH_AI_ENABLED'
Assert-EnabledFlag -Name 'BENETNASCH_AI_ARTICLE_INDEXING'
if ([string]::IsNullOrWhiteSpace($env:MEILI_MASTER_KEY)) {
    throw 'MEILI_MASTER_KEY must be set in .env.integration; the value is never printed.'
}

$repoRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot)).TrimEnd(
    [IO.Path]::DirectorySeparatorChar,
    [IO.Path]::AltDirectorySeparatorChar
)
$evidencePath = [IO.Path]::GetFullPath($QualityEvidencePath)
$repoPrefix = $repoRoot + [IO.Path]::DirectorySeparatorChar
if ($evidencePath.Equals($repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
    $evidencePath.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'QualityEvidencePath must be outside the repository.'
}
if (-not (Test-Path -LiteralPath $evidencePath -PathType Leaf)) {
    throw 'QualityEvidencePath does not exist.'
}
$evidenceItem = Get-Item -LiteralPath $evidencePath
if (($evidenceItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0 -or $evidenceItem.Length -le 0) {
    throw 'QualityEvidencePath must be a non-empty regular file, not a reparse point.'
}

try {
    $evidence = Get-Content -LiteralPath $evidencePath -Raw | ConvertFrom-Json
} catch {
    throw 'Quality evidence is not valid JSON.'
}

function Get-EvidenceValue {
    param([Parameter(Mandatory = $true)]$Object, [Parameter(Mandatory = $true)][string]$Name)

    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) {
        throw "Quality evidence is missing '$Name'."
    }
    return $property.Value
}

function Get-RequiredBoolean {
    param([Parameter(Mandatory = $true)]$Object, [Parameter(Mandatory = $true)][string]$Name)

    $value = Get-EvidenceValue -Object $Object -Name $Name
    if ($value -isnot [bool]) {
        throw "Quality evidence field '$Name' must be a JSON boolean."
    }
    return [bool]$value
}

if ([string](Get-EvidenceValue -Object $evidence -Name 'indexUid') -ne $CandidateIndex) {
    throw 'Quality evidence does not belong to the requested candidate index.'
}
$evidencePass = Get-RequiredBoolean -Object $evidence -Name 'pass'
$evidenceQualityPass = Get-RequiredBoolean -Object $evidence -Name 'qualityPass'
$evidenceSizeGatePass = Get-RequiredBoolean -Object $evidence -Name 'sizeGatePass'
if (-not $evidencePass -or -not $evidenceQualityPass -or -not $evidenceSizeGatePass) {
    throw 'Quality/P95 evidence did not pass all required gates.'
}
$documentCount = [int64](Get-EvidenceValue -Object $evidence -Name 'documentCount')
$evidenceMinimum = [int64](Get-EvidenceValue -Object $evidence -Name 'minimumDocumentCount')
$p95Milliseconds = [double](Get-EvidenceValue -Object $evidence -Name 'p95Milliseconds')
if ($documentCount -lt $MinimumDocumentCount -or $evidenceMinimum -lt $MinimumDocumentCount) {
    throw 'Quality evidence document count is below the requested representative minimum.'
}
if ([double]::IsNaN($p95Milliseconds) -or [double]::IsInfinity($p95Milliseconds) -or $p95Milliseconds -gt $TargetMilliseconds) {
    throw 'Quality evidence P95 exceeds the requested search latency target.'
}
try {
    $generatedAt = [DateTimeOffset]::Parse(
        [string](Get-EvidenceValue -Object $evidence -Name 'generatedAtUtc'),
        [Globalization.CultureInfo]::InvariantCulture,
        [Globalization.DateTimeStyles]::AssumeUniversal -bor [Globalization.DateTimeStyles]::AdjustToUniversal
    )
} catch {
    throw 'Quality evidence generatedAtUtc is invalid.'
}
if ($generatedAt -gt [DateTimeOffset]::UtcNow.AddMinutes(5) -or
    $generatedAt -lt [DateTimeOffset]::UtcNow.AddHours(-$MaximumEvidenceAgeHours)) {
    throw 'Quality evidence is outside the permitted freshness window.'
}

# A non-rename swap exchanges the contents of both UIDs.  Refuse an empty
# current index before invoking the write-capable backend command: swapping a
# populated candidate with an empty placeholder would make the configured
# candidate UID empty after the task succeeds.  First activation of a new
# versioned UID must deploy that UID directly; it is not an atomic swap.
$meiliBaseUrl = 'http://127.0.0.1:17700'
$meiliHeaders = @{ Authorization = "Bearer $env:MEILI_MASTER_KEY" }
foreach ($indexUid in @($CurrentIndex, $CandidateIndex)) {
    try {
        $indexMeta = Invoke-RestMethod -Method Get -Uri "$meiliBaseUrl/indexes/$indexUid" -Headers $meiliHeaders -TimeoutSec 10
        $indexStats = Invoke-RestMethod -Method Get -Uri "$meiliBaseUrl/indexes/$indexUid/stats" -Headers $meiliHeaders -TimeoutSec 10
    } catch {
        throw "Unable to inspect isolated Meilisearch index '$indexUid' before swap; no write request was sent."
    }
    if ([string]$indexMeta.primaryKey -ne 'id') {
        throw "Index '$indexUid' has primary key '$($indexMeta.primaryKey)', want 'id'; no write request was sent."
    }
    if ($null -eq $indexStats.numberOfDocuments) {
        throw "Index '$indexUid' returned no document count; no write request was sent."
    }
    $liveDocumentCount = [int64]$indexStats.numberOfDocuments
    if ($liveDocumentCount -lt 0) {
        throw "Index '$indexUid' returned an invalid document count; no write request was sent."
    }
    if ($indexUid -eq $CurrentIndex -and $liveDocumentCount -lt $CurrentMinimumDocumentCount) {
        throw "Current index '$CurrentIndex' has $liveDocumentCount document(s), below CurrentMinimumDocumentCount=$CurrentMinimumDocumentCount; a non-rename swap would make the populated candidate unavailable, so no write request was sent."
    }
    if ($indexUid -eq $CandidateIndex -and $liveDocumentCount -lt $MinimumDocumentCount) {
        throw "Candidate index '$CandidateIndex' has $liveDocumentCount document(s), below MinimumDocumentCount=$MinimumDocumentCount; no write request was sent."
    }
}

# Confirm the deployed binary exposes the guarded command before allowing the
# write.  The key is already inside the container; it is never passed to exec.
Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'backend', '/app/benetnasch', 'article-index', 'swap', '--help'
)

Write-Host "Submitting isolated article-index swap from $CurrentIndex to $CandidateIndex..." -ForegroundColor Cyan
Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'backend', '/app/benetnasch',
    'article-index', 'swap',
    '--current-index', $CurrentIndex,
    '--current-index-version', $CurrentIndexVersion,
    '--current-provider', $CurrentProvider,
    '--current-model', $CurrentModel,
    '--current-model-version', $CurrentModelVersion,
    '--current-dimension', $CurrentDimension,
    '--current-batch-size', $CurrentBatchSize,
    '--candidate-index', $CandidateIndex,
    '--allow-writes'
)

Write-Host 'Isolated article-index swap completed. Retain the old index and verify search again during the observation window.' -ForegroundColor Green
