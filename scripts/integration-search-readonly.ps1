param(
    [string]$MeiliBaseUrl = 'http://127.0.0.1:17700',
    [string]$IndexUid = 'article_chunks_v2',
    [ValidateSet('dataset.json', 'integration-dataset.json')]
    [string]$DatasetFileName = 'dataset.json',
    [int]$SamplesPerQuery = 3,
    [int]$TargetMilliseconds = 300,
    [int64]$MinimumDocumentCount = 0
)

$ErrorActionPreference = 'Stop'

# Read-only search checks use the same scoped environment loader as every other
# integration script.  This prevents a parent-shell key or target override from
# being used accidentally, and makes the documented command self-contained.
. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

function Assert-ReadOnlyMeiliUrl {
    param([Parameter(Mandatory = $true)][string]$Value)

    $uri = [Uri]$Value
    if ($uri.Scheme -ne 'http' -or $uri.Host -notin @('127.0.0.1', 'localhost') -or $uri.Port -ne 17700) {
        throw 'MeiliBaseUrl must point to the existing isolated loopback port 17700; this script refuses non-local targets.'
    }
    return $uri.AbsoluteUri.TrimEnd('/')
}

function Assert-StringSequence {
    param(
        [Parameter(Mandatory = $true)][object[]]$Actual,
        [Parameter(Mandatory = $true)][string[]]$Expected,
        [Parameter(Mandatory = $true)][string]$Name
    )

    if ($Actual.Count -ne $Expected.Count) {
        throw "$Name does not match the article chunk contract. Actual=$($Actual -join ',') Expected=$($Expected -join ',')"
    }
    for ($index = 0; $index -lt $Expected.Count; $index++) {
        if ([string]$Actual[$index] -ne $Expected[$index]) {
            throw "$Name does not match the article chunk contract. Actual=$($Actual -join ',') Expected=$($Expected -join ',')"
        }
    }
}

function Assert-StringSet {
    param(
        [Parameter(Mandatory = $true)][object[]]$Actual,
        [Parameter(Mandatory = $true)][string[]]$Expected,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $actualValues = @($Actual | ForEach-Object { [string]$_ } | Sort-Object)
    $expectedValues = @($Expected | ForEach-Object { [string]$_ } | Sort-Object)
    if ($actualValues.Count -ne $expectedValues.Count) {
        throw "$Name does not match the article chunk contract. Actual=$($Actual -join ',') Expected=$($Expected -join ',')"
    }
    for ($index = 0; $index -lt $expectedValues.Count; $index++) {
        if ($actualValues[$index] -ne $expectedValues[$index]) {
            throw "$Name does not match the article chunk contract. Actual=$($Actual -join ',') Expected=$($Expected -join ',')"
        }
    }
}

if ($SamplesPerQuery -lt 1 -or $SamplesPerQuery -gt 100) {
    throw 'SamplesPerQuery must be between 1 and 100.'
}
if ($TargetMilliseconds -le 0) {
    throw 'TargetMilliseconds must be positive.'
}
if ($MinimumDocumentCount -le 0) {
	throw 'MinimumDocumentCount must be positive; the benchmark requires a representative non-empty index.'
}
if ($IndexUid -notmatch '^article_chunks_[a-z0-9][a-z0-9_-]{0,30}$') {
    throw 'IndexUid must be a versioned article_chunks UID.'
}
if ([string]::IsNullOrWhiteSpace($env:MEILI_MASTER_KEY)) {
    throw 'MEILI_MASTER_KEY must be set; the value is never printed.'
}

$baseUrl = Assert-ReadOnlyMeiliUrl -Value $MeiliBaseUrl
$headers = @{
    Authorization = "Bearer $env:MEILI_MASTER_KEY"
    'Content-Type' = 'application/json'
}
$datasetRoot = Join-Path $PSScriptRoot '..\app\infra\search\evaldata'
$datasetPath = Join-Path $datasetRoot $DatasetFileName
$cases = @(Get-Content -Raw -LiteralPath $datasetPath | ConvertFrom-Json)
if ($cases.Count -eq 0) {
    throw "retrieval dataset is empty: $datasetPath"
}
if (@($cases | Where-Object { [string]$_.mode -ne 'keyword' }).Count -gt 0) {
    throw 'the read-only benchmark only accepts the current fixed keyword retrieval dataset.'
}

try {
    $index = Invoke-RestMethod -Method Get -Uri "$baseUrl/indexes/$IndexUid" -Headers $headers -TimeoutSec 10
} catch {
    $statusCode = $null
    if ($null -ne $_.Exception.Response) {
        $statusCode = [int]$_.Exception.Response.StatusCode
    }
    if ($statusCode -eq 404) {
        throw "candidate index '$IndexUid' is not provisioned; no write request was sent."
    }
    throw "unable to inspect candidate index '$IndexUid' (HTTP $statusCode): $($_.Exception.Message)"
}
if ([string]$index.primaryKey -ne 'id') {
    throw "candidate index '$IndexUid' has primary key '$($index.primaryKey)', want 'id'."
}
$stats = Invoke-RestMethod -Method Get -Uri "$baseUrl/indexes/$IndexUid/stats" -Headers $headers -TimeoutSec 10
if ($null -eq $stats.numberOfDocuments) {
    throw "candidate index '$IndexUid' returned no numberOfDocuments statistic."
}
$documentCount = [int64]$stats.numberOfDocuments
if ($documentCount -lt 0) {
    throw "candidate index '$IndexUid' returned an invalid negative document count."
}
if ($documentCount -lt $MinimumDocumentCount) {
    throw "candidate index '$IndexUid' has $documentCount document(s), below the required representative minimum of $MinimumDocumentCount; no benchmark search requests were sent."
}
$settings = Invoke-RestMethod -Method Get -Uri "$baseUrl/indexes/$IndexUid/settings" -Headers $headers -TimeoutSec 10
Assert-StringSequence -Actual @($settings.searchableAttributes) -Expected @('articleTitle', 'text') -Name 'searchable attributes'
Assert-StringSet -Actual @($settings.filterableAttributes) -Expected @('articleId', 'category', 'tags', 'publishedAt', 'publishedAtUnix', 'status', 'isDelete') -Name 'filterable attributes'

$searchCases = @()
$searchCases += @{
    attributesToRetrieve = @('id', 'articleId', 'articleTitle', 'text', 'category', 'tags', 'articleUrl', 'publishedAt', 'publishedAtUnix', 'chunkIndex', 'status', 'isDelete')
    attributesToSearchOn = @('articleTitle', 'text')
    filter = 'isDelete = 0 AND status = 1'
    limit = 8
}

$samples = [System.Collections.Generic.List[object]]::new()
foreach ($case in $cases) {
    for ($sampleIndex = 1; $sampleIndex -le $SamplesPerQuery; $sampleIndex++) {
        $body = $searchCases[0].Clone()
        $body.q = [string]$case.query
        $payload = $body | ConvertTo-Json -Depth 5 -Compress
        $stopwatch = [Diagnostics.Stopwatch]::StartNew()
        try {
            $response = Invoke-RestMethod -Method Post -Uri "$baseUrl/indexes/$IndexUid/search" -Headers $headers -Body $payload -TimeoutSec 30
        } catch {
            throw "read-only retrieval case '$($case.id)' failed: $($_.Exception.Message)"
        } finally {
            $stopwatch.Stop()
        }
        $hitObjects = @($response.hits | Where-Object { $null -ne $_ })
        foreach ($hit in $hitObjects) {
            if ([int]$hit.status -ne 1 -or [int]$hit.isDelete -ne 0) {
                throw "read-only retrieval case '$($case.id)' returned a non-public hit; the candidate index failed the visibility gate."
            }
        }
        $hitIds = @($hitObjects | ForEach-Object { [string]$_.id } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique)
        $expectedIds = @($case.expected | ForEach-Object { [string]$_ } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique)
        $relevant = @($hitIds | Where-Object { $_ -in $expectedIds }).Count
        $expectNoResult = [bool]$case.expectNoResult
        $recallAt8 = 0.0
        $reciprocalRank = 0.0
        $noResultCorrect = $false
        if ($expectNoResult -and $hitIds.Count -ne 0) {
            throw "read-only retrieval case '$($case.id)' expected no result but returned $($hitIds.Count) hit(s)."
        }
        if (-not $expectNoResult -and $relevant -eq 0) {
            throw "read-only retrieval case '$($case.id)' returned no expected public chunk."
        }
        if ($expectNoResult) {
            $noResultCorrect = $hitIds.Count -eq 0
        } else {
            $recallAt8 = [double]$relevant / [double]$expectedIds.Count
            for ($rank = 0; $rank -lt $hitIds.Count; $rank++) {
                if ($hitIds[$rank] -in $expectedIds) {
                    $reciprocalRank = 1.0 / [double]($rank + 1)
                    break
                }
            }
        }
        $samples.Add([pscustomobject]@{
                Case       = [string]$case.id
                Sample     = $sampleIndex
                DurationMs = [Math]::Round($stopwatch.Elapsed.TotalMilliseconds, 2)
                Hits       = $hitIds.Count
                Relevant   = $relevant
                RecallAt8  = $recallAt8
                ReciprocalRank = $reciprocalRank
                NoResultExpected = $expectNoResult
                NoResultCorrect = $noResultCorrect
            })
    }
}

$durations = @($samples | ForEach-Object { [double]$_.DurationMs } | Sort-Object)
function Get-Percentile {
    param([double[]]$Values, [int]$Percentile)

    $index = [Math]::Max(0, [Math]::Ceiling($Values.Count * $Percentile / 100.0) - 1)
    return $Values[$index]
}

$p50 = Get-Percentile -Values $durations -Percentile 50
$p95 = Get-Percentile -Values $durations -Percentile 95
$max = $durations[-1]
$qualityCases = @($samples | Group-Object -Property Case | ForEach-Object { $_.Group[0] })
$positiveQualityCases = @($qualityCases | Where-Object { -not $_.NoResultExpected })
$noResultQualityCases = @($qualityCases | Where-Object { $_.NoResultExpected })
$recallAt8 = if ($positiveQualityCases.Count -gt 0) {
    [Math]::Round((($positiveQualityCases | Measure-Object -Property RecallAt8 -Average).Average), 6)
} else {
    0.0
}
$mrr = if ($positiveQualityCases.Count -gt 0) {
    [Math]::Round((($positiveQualityCases | Measure-Object -Property ReciprocalRank -Average).Average), 6)
} else {
    0.0
}
$noResultAccuracy = if ($noResultQualityCases.Count -gt 0) {
    [Math]::Round((($noResultQualityCases | Where-Object { $_.NoResultCorrect }).Count / [double]$noResultQualityCases.Count), 6)
} else {
    0.0
}
$qualityPass = $positiveQualityCases.Count -gt 0 -and $noResultQualityCases.Count -gt 0 -and
    ($noResultQualityCases | Where-Object { -not $_.NoResultCorrect }).Count -eq 0 -and
    ($positiveQualityCases | Where-Object { $_.Relevant -eq 0 }).Count -eq 0
$sizeGateConfigured = $true
$sizeGatePass = $documentCount -ge $MinimumDocumentCount
$report = [ordered]@{
    generatedAtUtc   = (Get-Date).ToUniversalTime().ToString('o')
    indexUid         = $IndexUid
    documentCount    = $documentCount
    minimumDocumentCount = $MinimumDocumentCount
    sizeGateConfigured = $sizeGateConfigured
    sizeGatePass     = $sizeGatePass
    queries          = $cases.Count
    positiveQueries  = $positiveQualityCases.Count
    noResultQueries  = $noResultQualityCases.Count
    recallAt8        = $recallAt8
    mrr              = $mrr
    noResultAccuracy = $noResultAccuracy
    samples          = $durations.Count
    samplesPerQuery  = $SamplesPerQuery
    p50Milliseconds  = $p50
    p95Milliseconds  = $p95
    maxMilliseconds   = $max
    targetMilliseconds = $TargetMilliseconds
    qualityPass       = $qualityPass
    pass             = $qualityPass -and $p95 -le $TargetMilliseconds -and $sizeGatePass
    cases            = @($samples)
}

$report | ConvertTo-Json -Depth 6
if (-not $report.pass) {
    exit 2
}
