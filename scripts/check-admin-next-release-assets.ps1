[CmdletBinding()]
param(
    [string]$AdminNextDist = 'web/admin-next/dist',
    [string]$LegacyAdminDist = 'web/admin/dist'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$repoRoot = (Get-Item -LiteralPath $repoRoot).FullName
$repoPrefix = $repoRoot.TrimEnd([char[]]"/\") + [System.IO.Path]::DirectorySeparatorChar

function Resolve-RepoPath {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Label
    )

    $candidate = if ([System.IO.Path]::IsPathRooted($Path)) {
        $Path
    } else {
        Join-Path $repoRoot $Path
    }
    $fullPath = [System.IO.Path]::GetFullPath($candidate)
    if (-not ($fullPath.Equals($repoRoot, [System.StringComparison]::OrdinalIgnoreCase) -or
            $fullPath.StartsWith($repoPrefix, [System.StringComparison]::OrdinalIgnoreCase))) {
        throw "$Label must stay inside the repository: $Path"
    }
    return $fullPath
}

function Get-RelativeRepoPath {
    param([Parameter(Mandatory = $true)][string]$Path)

    return $Path.Substring($repoPrefix.Length).Replace([System.IO.Path]::DirectorySeparatorChar, '/')
}

function Get-RequiredFile {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Label
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Label does not exist: $Path"
    }
    $file = Get-Item -LiteralPath $Path
    if ($file.Length -le 0) {
        throw "$Label is empty: $Path"
    }
    return $file
}

function Assert-PathInside {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Root,
        [Parameter(Mandatory = $true)][string]$Label
    )

    $rootPrefix = $Root.TrimEnd([char[]]"/\") + [System.IO.Path]::DirectorySeparatorChar
    if (-not ($Path.Equals($Root, [System.StringComparison]::OrdinalIgnoreCase) -or
            $Path.StartsWith($rootPrefix, [System.StringComparison]::OrdinalIgnoreCase))) {
        throw "$Label escapes its distribution directory: $Path"
    }
}

$adminNextDistPath = Resolve-RepoPath -Path $AdminNextDist -Label 'Admin-next distribution path'
$legacyAdminDistPath = Resolve-RepoPath -Path $LegacyAdminDist -Label 'Legacy admin distribution path'

if (-not (Test-Path -LiteralPath $adminNextDistPath -PathType Container)) {
    throw "Admin-next distribution directory does not exist: $adminNextDistPath"
}
if (-not (Test-Path -LiteralPath $legacyAdminDistPath -PathType Container)) {
    throw "Legacy admin distribution directory does not exist: $legacyAdminDistPath"
}

$adminNextIndex = Get-RequiredFile -Path (Join-Path $adminNextDistPath 'index.html') -Label 'Admin-next index.html'
$legacyAdminIndex = Get-RequiredFile -Path (Join-Path $legacyAdminDistPath 'index.html') -Label 'Legacy admin rollback index.html'
$indexContent = Get-Content -LiteralPath $adminNextIndex.FullName -Raw

if ($indexContent -notmatch '(?i)/assets/') {
    throw 'Admin-next index.html does not look like a production Vite build: /assets/ reference is missing.'
}
if ($indexContent -match '(?i)/src/main\.ts') {
    throw 'Admin-next index.html still points at a Vite development entrypoint.'
}

$attributePattern = '(?i)(?:src|href)\s*=\s*"([^"]+)"'
$references = @(
    [System.Text.RegularExpressions.Regex]::Matches($indexContent, $attributePattern) |
        ForEach-Object { $_.Groups[1].Value } |
        Sort-Object -Unique
)
if ($references.Count -eq 0) {
    throw 'Admin-next index.html contains no local asset references.'
}

$resolvedAssets = @()
foreach ($reference in $references) {
    $assetReference = ($reference -split '[?#]', 2)[0]
    if ([string]::IsNullOrWhiteSpace($assetReference) -or
        $assetReference -match '^(?i)(?:[a-z][a-z0-9+.-]*:|//|#)') {
        continue
    }

    if ($assetReference.StartsWith('/')) {
        $assetReference = $assetReference.Substring(1)
    }
    $assetReference = [System.Uri]::UnescapeDataString($assetReference)
    if ($assetReference -match '(^|/)\.\.(?:/|$)') {
        throw "Admin-next asset reference contains a parent traversal: $reference"
    }

    $assetPath = [System.IO.Path]::GetFullPath((Join-Path $adminNextDistPath $assetReference))
    Assert-PathInside -Path $assetPath -Root $adminNextDistPath -Label 'Admin-next asset reference'
    $assetFile = Get-RequiredFile -Path $assetPath -Label "Admin-next asset '$reference'"
    $resolvedAssets += $assetFile
}

if ($resolvedAssets.Count -eq 0) {
    throw 'Admin-next index.html contains no local assets that can be verified.'
}

$applicationChunks = @(
    $resolvedAssets |
        Where-Object { $_.Extension -ieq '.js' -and $_.BaseName -notlike 'vendor-*' } |
        Sort-Object FullName -Unique
)
$forbiddenApplicationPatterns = @(
    '(?i)https?://(?:localhost|127\.0\.0\.1)(?::\d+)?',
    '(?i)(?<![A-Za-z0-9_])(?:localhost|127\.0\.0\.1)(?::\d+)?',
    '(?i):8082\b',
    '(?i)VITE_(?:ADMIN_API_TARGET|API_TARGET)',
    '(?i)/src/main\.ts'
)
foreach ($chunk in $applicationChunks) {
    $chunkContent = Get-Content -LiteralPath $chunk.FullName -Raw
    foreach ($pattern in $forbiddenApplicationPatterns) {
        if ($chunkContent -match $pattern) {
            throw "Admin-next application chunk contains a development-only target ($pattern): $($chunk.Name)"
        }
    }
}

$allFiles = @(Get-ChildItem -LiteralPath $adminNextDistPath -Recurse -File)
$totalBytes = [int64](($allFiles | Measure-Object -Property Length -Sum).Sum)
$indexHash = (Get-FileHash -LiteralPath $adminNextIndex.FullName -Algorithm SHA256).Hash

$manifest = [ordered]@{
    adminNext = [ordered]@{
        distribution = Get-RelativeRepoPath -Path $adminNextDistPath
        index = Get-RelativeRepoPath -Path $adminNextIndex.FullName
        files = $allFiles.Count
        bytes = $totalBytes
        indexSha256 = $indexHash
        verifiedReferences = $resolvedAssets.Count
        verifiedApplicationChunks = $applicationChunks.Count
    }
    rollback = [ordered]@{
        distribution = Get-RelativeRepoPath -Path $legacyAdminDistPath
        index = Get-RelativeRepoPath -Path $legacyAdminIndex.FullName
        indexBytes = [int64]$legacyAdminIndex.Length
        readyForSwitch = $false
    }
    sideEffects = 'read-only'
}

$manifest | ConvertTo-Json -Depth 5
