[CmdletBinding()]
param(
    [string]$BaseUrl = 'http://127.0.0.1:18018',
    [Parameter(Mandatory = $true)]
    [string]$FixtureDirectory,
    [string]$OutputPath = '',
    [string]$CaseId,
    [string]$FixtureFileName,
    [string]$FixtureManifestPath = '',
    [ValidateSet('1', '2')]
    [string]$SampleSetIndex = '1',
    [switch]$PreflightOnly,
    [switch]$AllowExternalProviderCalls,
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

# Normal execution sends licensed image fixtures to the isolated Vision
# endpoint and creates pending review records. PreflightOnly only reads local
# fixture metadata/signatures and never needs credentials or either write gate.
if ($PreflightOnly) {
    # Local fixture validation intentionally has no external side effects.
} elseif (-not $AllowExternalProviderCalls -or -not $AllowWrites) {
    throw 'Refusing isolated Vision evaluation. Re-run with -AllowExternalProviderCalls -AllowWrites during an approved integration window.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

function Assert-EnabledFlag {
    param([Parameter(Mandatory = $true)][string]$Name)

    $parsed = $false
    $raw = [Environment]::GetEnvironmentVariable($Name, 'Process')
    if (-not [bool]::TryParse($raw, [ref]$parsed) -or -not $parsed) {
        throw "$Name must be explicitly set to true in .env.integration; the script never changes rollout flags."
    }
}

function Resolve-ExternalPath {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][ValidateSet('Leaf', 'Container')][string]$PathType
    )

    try {
        $resolved = Resolve-Path -LiteralPath $Path -ErrorAction Stop
    } catch {
        throw "Evaluation path does not exist: $Path"
    }
    if (-not (Test-Path -LiteralPath $resolved.Path -PathType $PathType)) {
        throw "Evaluation path has the wrong type: $Path"
    }

    $item = Get-Item -LiteralPath $Path -Force
    if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "Evaluation path must not be a symlink or reparse point: $Path"
    }

    $fullPath = [IO.Path]::GetFullPath($resolved.Path)
    $repoPath = [IO.Path]::GetFullPath($script:IntegrationRepoRoot)
    if ($fullPath.Equals($repoPath, [StringComparison]::OrdinalIgnoreCase) -or
        $fullPath.StartsWith($repoPath + '\', [StringComparison]::OrdinalIgnoreCase) -or
        $fullPath.StartsWith($repoPath + '/', [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Vision fixtures and output must be outside the repository; do not commit images or model output.'
    }
    if ($fullPath -match '(?i)(^|[\\/])\.codebuddy([\\/]|$)') {
        throw 'Vision fixtures and output must not use the .codebuddy evaluation directory.'
    }
    return $fullPath
}

function Assert-ImageSignature {
    param(
        [Parameter(Mandatory = $true)][byte[]]$Bytes,
        [Parameter(Mandatory = $true)][string]$MIMEType,
        [Parameter(Mandatory = $true)][string]$CaseID
    )

    $matches = switch ($MIMEType) {
        'image/jpeg' {
            $Bytes.Length -ge 3 -and [int]$Bytes[0] -eq 0xFF -and [int]$Bytes[1] -eq 0xD8 -and [int]$Bytes[2] -eq 0xFF
        }
        'image/png' {
            $signature = [byte[]](0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A)
            $Bytes.Length -ge $signature.Length -and (@(0..($signature.Length - 1) | Where-Object { $Bytes[$_] -ne $signature[$_] }).Count -eq 0)
        }
        'image/gif' {
            $Bytes.Length -ge 6 -and [Text.Encoding]::ASCII.GetString($Bytes, 0, 6) -in @('GIF87a', 'GIF89a')
        }
        'image/webp' {
            $Bytes.Length -ge 12 -and [Text.Encoding]::ASCII.GetString($Bytes, 0, 4) -eq 'RIFF' -and [Text.Encoding]::ASCII.GetString($Bytes, 8, 4) -eq 'WEBP'
        }
        default { $false }
    }
    if (-not $matches) {
        throw "Vision fixture '$CaseID' does not match its declared image MIME type."
    }
}

if (-not $PreflightOnly) {
    Assert-EnabledFlag -Name 'BENETNASCH_AI_ENABLED'
    Assert-EnabledFlag -Name 'BENETNASCH_AI_VISION'
    if ([string]::IsNullOrWhiteSpace($env:OPENAI_API_KEY)) {
        throw 'OPENAI_API_KEY must be set in .env.integration for the configured DeepSeek Vision route; the value is never printed.'
    }
}

function Get-AdminToken {
    if ([string]::IsNullOrWhiteSpace($env:E2E_ADMIN_EMAIL) -or [string]::IsNullOrWhiteSpace($env:E2E_ADMIN_PASSWORD)) {
        throw 'isolated admin credentials must be set in .env.integration; credentials are never printed.'
    }

    $body = @{ username = $env:E2E_ADMIN_EMAIL; password = $env:E2E_ADMIN_PASSWORD } | ConvertTo-Json -Compress
    $response = Invoke-IntegrationRequest -Uri "$script:ApiBase/api/users/login" -Method POST -Body $body
    if ($response.StatusCode -ne 200) {
        throw "isolated admin login failed with HTTP $($response.StatusCode)."
    }
    try {
        $payload = $response.Content | ConvertFrom-Json
    } catch {
        throw 'isolated admin login returned malformed JSON.'
    }
    if ($null -eq $payload -or $payload.flag -ne $true -or [int]$payload.code -ne 20000) {
        throw 'isolated admin login returned an unsuccessful result.'
    }
    if ($payload.data -is [string] -and -not [string]::IsNullOrWhiteSpace([string]$payload.data)) {
        return [string]$payload.data
    }
    foreach ($field in @('token', 'accessToken', 'access_token')) {
        $property = $payload.data.PSObject.Properties[$field]
        if ($null -ne $property -and -not [string]::IsNullOrWhiteSpace([string]$property.Value)) {
            return [string]$property.Value
        }
    }
    throw 'isolated admin login did not return a token.'
}

try {
    $baseURI = [Uri]$BaseUrl
} catch {
    throw 'BaseUrl is not a valid URI.'
}
if ($baseURI.Scheme -ne 'http' -or $baseURI.Host -notin @('127.0.0.1', 'localhost') -or $baseURI.Port -ne 18018) {
    throw 'BaseUrl must point to isolated admin-next loopback port 18018; refusing another target.'
}
$script:ApiBase = $baseURI.AbsoluteUri.TrimEnd('/')

$fixtureRoot = Resolve-ExternalPath -Path $FixtureDirectory -PathType Container

$datasetPath = Join-Path $script:IntegrationRepoRoot 'app\infra\ai\evaldata\vision_dataset.json'
try {
    $cases = @(Get-Content -Raw -LiteralPath $datasetPath | ConvertFrom-Json)
} catch {
    throw 'Vision evaluation dataset is not valid JSON.'
}
if ($cases.Count -eq 0) {
    throw 'Vision evaluation dataset is empty.'
}

$CaseId = if ($null -eq $CaseId) { '' } else { $CaseId.Trim() }
$FixtureFileName = if ($null -eq $FixtureFileName) { '' } else { $FixtureFileName.Trim() }
$FixtureManifestPath = if ($null -eq $FixtureManifestPath) { '' } else { $FixtureManifestPath.Trim() }
if (($CaseId -eq '') -ne ($FixtureFileName -eq '')) {
    throw 'CaseId and FixtureFileName must be provided together for single-case Vision evaluation.'
}
if ($FixtureManifestPath -ne '' -and $FixtureFileName -ne '') {
    throw 'FixtureManifestPath and FixtureFileName cannot be used together; choose one explicit fixture selector.'
}
if ($CaseId -ne '') {
    if ($CaseId -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]{0,119}$') {
        throw 'CaseId contains unsupported characters or exceeds 120 bytes.'
    }
    $cases = @($cases | Where-Object { [string]$_.id -eq $CaseId })
    if ($cases.Count -ne 1) {
        throw "Vision evaluation case '$CaseId' is not present exactly once in the fixed dataset."
    }
    if ($FixtureFileName -notmatch '^[^\\/:*?"<>|]+$' -or $FixtureFileName -eq '.' -or $FixtureFileName -eq '..') {
        throw 'FixtureFileName must be a single file name without path separators or traversal components.'
    }
}

$extensionsByMIME = @{
    'image/jpeg' = @('.jpg', '.jpeg')
    'image/png'  = @('.png')
    'image/gif'  = @('.gif')
    'image/webp' = @('.webp')
}
$maxFixtureBytes = 4MB
$preparedCases = [System.Collections.Generic.List[object]]::new()
$fixtureManifest = $null
if ($FixtureManifestPath -ne '') {
    $manifestPath = Resolve-ExternalPath -Path $FixtureManifestPath -PathType Leaf
    try {
        $manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
    } catch {
        throw 'FixtureManifestPath must contain a valid JSON object mapping fixtureId to a direct file name.'
    }
    if ($null -eq $manifest -or $manifest -is [array]) {
        throw 'FixtureManifestPath must contain a JSON object mapping fixtureId to a direct file name.'
    }
    $fixtureManifest = @{}
    foreach ($property in $manifest.PSObject.Properties) {
        $fixtureID = [string]$property.Name
        $fileName = [string]$property.Value
        if ($fixtureID -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]{0,119}$' -or
            $property.Value -isnot [string] -or
            $fileName -notmatch '^[^\\/:*?"<>|]+$' -or
            $fileName -eq '.' -or $fileName -eq '..') {
            throw "Fixture manifest entry '$fixtureID' must map to one direct file name."
        }
        $fixtureManifest[$fixtureID] = $fileName
    }
    if ($fixtureManifest.Count -eq 0) {
        throw 'FixtureManifestPath must contain at least one fixture mapping.'
    }
}

function Resolve-FixtureFile {
    param(
        [Parameter(Mandatory = $true)][string]$FixtureRoot,
        [Parameter(Mandatory = $true)][string]$FixtureID,
        [Parameter(Mandatory = $true)][string]$FileName
    )

    if ($FileName -notmatch '^[^\\/:*?"<>|]+$' -or $FileName -eq '.' -or $FileName -eq '..') {
        throw "Vision fixture '$FixtureID' must be a direct file name without path separators."
    }
    $candidate = Join-Path $FixtureRoot $FileName
    if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
        throw "Missing authorized Vision fixture '$FixtureID' ($FileName)."
    }
    $resolved = Resolve-ExternalPath -Path $candidate -PathType Leaf
    $fixtureRootPrefix = "$FixtureRoot\"
    if (-not $resolved.StartsWith($fixtureRootPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Vision fixture '$FixtureID' must resolve directly below FixtureDirectory."
    }
    return $resolved
}

function Get-ImageMIMETypeFromFileName {
    param([Parameter(Mandatory = $true)][string]$FileName)

    switch ([IO.Path]::GetExtension($FileName).ToLowerInvariant()) {
        '.jpg' { return 'image/jpeg' }
        '.jpeg' { return 'image/jpeg' }
        '.png' { return 'image/png' }
        '.gif' { return 'image/gif' }
        '.webp' { return 'image/webp' }
        default { return '' }
    }
}

function Resolve-ChineseSampleFileName {
    param(
        [Parameter(Mandatory = $true)][string]$Category,
        [Parameter(Mandatory = $true)][string]$Index
    )

    $prefixByCategory = @{
        'scene'     = '普通'
        'ocr'       = '文字'
        'chart'      = '图表'
        'ambiguity' = '歧义'
        'safety'    = '恶意'
        'privacy'   = '隐私'
    }
    if (-not $prefixByCategory.ContainsKey($Category)) {
        return ''
    }
    return "$($prefixByCategory[$Category])$Index.jpg"
}

# Resolve and validate every fixture before obtaining a token or calling the
# API. A missing/invalid later case must never leave a partially written run.
foreach ($case in $cases) {
    $caseID = [string]$case.id
    $fixtureID = [string]$case.fixtureId
    $mimeType = ([string]$case.mimeType).ToLowerInvariant()
    if ($caseID -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]{0,119}$' -or
        $fixtureID -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]{0,119}$' -or
        -not $extensionsByMIME.ContainsKey($mimeType) -or
        [string]::IsNullOrWhiteSpace([string]$case.prompt) -or
        ("vision-eval-$caseID").Length -gt 128) {
        throw "Vision evaluation case '$caseID' has an invalid contract."
    }

    $fixturePath = $null
    $mappedFileName = $null
    $isChineseSample = $false
    if ($null -ne $fixtureManifest) {
        if (-not $fixtureManifest.ContainsKey($fixtureID)) {
            throw "FixtureManifestPath is missing a mapping for '$fixtureID'."
        }
        $mappedFileName = [string]$fixtureManifest[$fixtureID]
    }
    if ($FixtureFileName -ne '') {
        $fixturePath = Resolve-FixtureFile -FixtureRoot $fixtureRoot -FixtureID $fixtureID -FileName $FixtureFileName
    } elseif ($null -ne $fixtureManifest) {
        $fixturePath = Resolve-FixtureFile -FixtureRoot $fixtureRoot -FixtureID $fixtureID -FileName $mappedFileName
    } else {
        $sampleFileName = Resolve-ChineseSampleFileName -Category ([string]$case.category) -Index $SampleSetIndex
        if ($sampleFileName -ne '' -and (Test-Path -LiteralPath (Join-Path $fixtureRoot $sampleFileName) -PathType Leaf)) {
            $fixturePath = Resolve-FixtureFile -FixtureRoot $fixtureRoot -FixtureID $fixtureID -FileName $sampleFileName
            $isChineseSample = $true
        }
        foreach ($extension in @($extensionsByMIME[$mimeType])) {
            if ($null -eq $fixturePath) {
                $candidate = Join-Path $fixtureRoot ($fixtureID + $extension)
                if (Test-Path -LiteralPath $candidate -PathType Leaf) {
                    if ($null -ne $fixturePath) {
                        throw "Vision fixture '$fixtureID' has more than one matching extension."
                    }
                    $fixturePath = Resolve-ExternalPath -Path $candidate -PathType Leaf
                }
            }
        }
    }
    if ($null -eq $fixturePath) {
        throw "Missing authorized Vision fixture '$fixtureID' ($mimeType)."
    }

    $bytes = [IO.File]::ReadAllBytes($fixturePath)
    if ($bytes.Length -le 0 -or $bytes.Length -gt $maxFixtureBytes) {
        throw "Vision fixture '$fixtureID' must be between 1 byte and 4 MiB."
    }
    $effectiveMimeType = $mimeType
    if ($isChineseSample) {
        $effectiveMimeType = Get-ImageMIMETypeFromFileName -FileName ([IO.Path]::GetFileName($fixturePath))
        if ($effectiveMimeType -eq '') {
            throw "Vision fixture '$fixtureID' has an unsupported Chinese sample extension."
        }
    }
    Assert-ImageSignature -Bytes $bytes -MIMEType $effectiveMimeType -CaseID $caseID
    $preparedCases.Add([pscustomobject]@{
        Case        = $case
        CaseID      = $caseID
        FixtureID   = $fixtureID
        MimeType    = $effectiveMimeType
        FixturePath = $fixturePath
    })
}

Write-Host "Validated $($preparedCases.Count) Vision fixture(s) before any API call." -ForegroundColor Green
if ($PreflightOnly) {
    Write-Host 'Vision fixture preflight completed without credentials, API calls, or review writes.' -ForegroundColor Green
    return
}

if ([string]::IsNullOrWhiteSpace($OutputPath)) {
    throw 'OutputPath is required for a real Vision evaluation; use -PreflightOnly for a read-only fixture check.'
}

$outputFullPath = [IO.Path]::GetFullPath($OutputPath)
$outputParent = Split-Path -Parent $outputFullPath
if ([string]::IsNullOrWhiteSpace($outputParent) -or -not (Test-Path -LiteralPath $outputParent -PathType Container)) {
    throw 'OutputPath parent must already exist; the script will not create evaluation directories.'
}
$null = Resolve-ExternalPath -Path $outputParent -PathType Container
if (Test-Path -LiteralPath $outputFullPath) {
    throw 'OutputPath already exists; use a new output file so an evaluation run cannot overwrite evidence.'
}

$records = [System.Collections.Generic.List[object]]::new()
$token = Get-AdminToken
$headers = @{ Authorization = "Bearer $token" }

foreach ($preparedCase in $preparedCases) {
    $case = $preparedCase.Case
    $caseID = [string]$preparedCase.CaseID
    $fixtureID = [string]$preparedCase.FixtureID
    $mimeType = [string]$preparedCase.MimeType
    $bytes = [IO.File]::ReadAllBytes([string]$preparedCase.FixturePath)
    if ($bytes.Length -le 0 -or $bytes.Length -gt $maxFixtureBytes) {
        throw "Vision fixture '$fixtureID' must be between 1 byte and 4 MiB."
    }
    Assert-ImageSignature -Bytes $bytes -MIMEType $mimeType -CaseID $caseID

    $requestBody = [ordered]@{
        targetType  = 'draft'
        targetId    = "vision-eval-$caseID"
        prompt      = [string]$case.prompt
        imageBase64 = [Convert]::ToBase64String($bytes)
        mimeType    = $mimeType
        detail      = 'auto'
    } | ConvertTo-Json -Depth 5 -Compress
    $response = Invoke-IntegrationRequest `
        -Uri "$script:ApiBase/api/admin/ai/vision/preview" `
        -Method POST `
        -Headers $headers `
        -Body $requestBody
    if ($response.StatusCode -ne 200) {
        throw "Vision evaluation case '$caseID' returned HTTP $($response.StatusCode)."
    }
    try {
        $payload = $response.Content | ConvertFrom-Json
    } catch {
        throw "Vision evaluation case '$caseID' returned malformed JSON."
    }
    if ($null -eq $payload -or $payload.flag -ne $true -or [int]$payload.code -ne 20000) {
        throw "Vision evaluation case '$caseID' was rejected by the isolated API."
    }
    $result = $payload.data
    $runID = [string]$result.runId
    $reviewID = [string]$result.reviewId
    $preview = [string]$result.preview
    if ($result.operation -ne 'vision' -or
        $runID -notmatch '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' -or
        $reviewID -notmatch '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' -or
        [string]::IsNullOrWhiteSpace($preview)) {
        throw "Vision evaluation case '$caseID' returned an incomplete preview result."
    }

    # Keep prompt/image bytes out of the evidence artifact. The preview is
    # intentionally retained outside the repository so two independent raters
    # can score the exact pending result later.
    $records.Add([ordered]@{
        schemaVersion = 1
        generatedAtUtc = (Get-Date).ToUniversalTime().ToString('o')
        caseId         = $caseID
        category       = [string]$case.category
        fixtureId      = $fixtureID
        mimeType       = $mimeType
        reviewId       = $reviewID
        runId          = $runID
        preview        = $preview
    })
    Write-Host "Vision evaluation case '$caseID' completed; preview retained outside the repository." -ForegroundColor Green
}

$lines = [System.Collections.Generic.List[string]]::new()
foreach ($record in $records) {
    $lines.Add(($record | ConvertTo-Json -Depth 5 -Compress))
}
[IO.File]::WriteAllText(
    $outputFullPath,
    (($lines -join [Environment]::NewLine) + [Environment]::NewLine),
    [Text.UTF8Encoding]::new($false)
)
Write-Host "Isolated Vision evaluation captured $($records.Count) case(s) at '$outputFullPath'." -ForegroundColor Green
Write-Host 'The API created pending review records only; human scoring and any approval remain separate operations.' -ForegroundColor Yellow
