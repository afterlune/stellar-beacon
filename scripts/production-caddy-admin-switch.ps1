[CmdletBinding()]
param(
    [ValidateSet('Plan', 'Apply', 'VerifyRetention')]
    [string]$Action = 'Plan',
    [Parameter(Mandatory = $true)]
    [string]$StaticRoot,
    [string]$ArchiveRoot,
    [string]$SourceDist = 'web/admin-next/dist',
    [string]$ReleaseId,
    [string]$ReleaseTicket,
    [string]$ExpectedIndexSha256,
    [ValidateRange(1, 8760)]
    [int]$RetentionHours = 168,
    [switch]$AllowProductionSwitch,
    [switch]$ConfirmProductionTarget
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

function Resolve-ExternalPath {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Label
    )

    $fullPath = [System.IO.Path]::GetFullPath($Path)
    if ($fullPath.Equals($repoRoot, [System.StringComparison]::OrdinalIgnoreCase) -or
            $fullPath.StartsWith($repoPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "$Label must be outside the repository: $Path"
    }
    $root = [System.IO.Path]::GetPathRoot($fullPath)
    if ($fullPath.Equals($root, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "$Label must not be a filesystem root: $Path"
    }
    return $fullPath.TrimEnd([char[]]"/\")
}

function Assert-ReleaseId {
    param([Parameter(Mandatory = $true)][string]$Value)

    if ($Value -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$') {
        throw 'ReleaseId must contain only letters, numbers, dot, underscore, or hyphen and be at most 64 characters.'
    }
}

function Assert-ReleaseTicket {
    param([Parameter(Mandatory = $true)][string]$Value)

    if ($Value -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$') {
        throw 'ReleaseTicket must contain only letters, numbers, dot, underscore, or hyphen and be at most 128 characters.'
    }
}

function Assert-Sha256 {
    param(
        [Parameter(Mandatory = $true)][string]$Value,
        [Parameter(Mandatory = $true)][string]$Label
    )

    if ($Value -notmatch '^[A-Fa-f0-9]{64}$') {
        throw "$Label must be a 64-character SHA-256 hex digest."
    }
}

function Get-RequiredDirectory {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Label
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
        throw "$Label does not exist: $Path"
    }
    $item = Get-Item -LiteralPath $Path
    if ($item.Attributes -band [System.IO.FileAttributes]::ReparsePoint) {
        throw "$Label must not be a symbolic link or reparse point: $Path"
    }
    return $item
}

function Get-RequiredFile {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Label
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Label does not exist: $Path"
    }
    $item = Get-Item -LiteralPath $Path
    if ($item.Length -le 0) {
        throw "$Label is empty: $Path"
    }
    if ($item.Attributes -band [System.IO.FileAttributes]::ReparsePoint) {
        throw "$Label must not be a symbolic link or reparse point: $Path"
    }
    return $item
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
        throw "$Label escapes its allowed directory: $Path"
    }
}

function Assert-NoReparsePoints {
    param(
        [Parameter(Mandatory = $true)][string]$Root,
        [Parameter(Mandatory = $true)][string]$Label
    )

    foreach ($item in @(Get-ChildItem -LiteralPath $Root -Force -Recurse)) {
        if ($item.Attributes -band [System.IO.FileAttributes]::ReparsePoint) {
            throw "$Label contains a symbolic link or reparse point: $($item.FullName)"
        }
    }
}

function Get-RelativePath {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Root
    )

    $prefix = $Root.TrimEnd([char[]]"/\") + [System.IO.Path]::DirectorySeparatorChar
    Assert-PathInside -Path $Path -Root $Root -Label 'relative path'
    return $Path.Substring($prefix.Length).Replace([System.IO.Path]::DirectorySeparatorChar, '/')
}

function Assert-AdminBuild {
    param([Parameter(Mandatory = $true)][string]$Path)

    $index = Get-RequiredFile -Path (Join-Path $Path 'index.html') -Label 'admin-next index.html'
    $content = Get-Content -LiteralPath $index.FullName -Raw
    if ($content -notmatch '(?i)/assets/') {
        throw 'admin-next index.html does not look like a production Vite build: /assets/ reference is missing.'
    }
    if ($content -match '(?i)/src/main\.ts') {
        throw 'admin-next index.html still points at a development entrypoint.'
    }

    $references = @(
        [System.Text.RegularExpressions.Regex]::Matches($content, '(?i)(?:src|href)\s*=\s*"([^"]+)"') |
            ForEach-Object { $_.Groups[1].Value } |
            Sort-Object -Unique
    )
    if ($references.Count -eq 0) {
        throw 'admin-next index.html contains no local asset references.'
    }

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
            throw "admin-next asset reference contains parent traversal: $reference"
        }
        $assetPath = [System.IO.Path]::GetFullPath((Join-Path $Path $assetReference))
        Assert-PathInside -Path $assetPath -Root $Path -Label 'admin-next asset reference'
        [void](Get-RequiredFile -Path $assetPath -Label "admin-next asset '$reference'")
    }
    Assert-NoReparsePoints -Root $Path -Label 'admin-next build'
    return $index
}

function Get-IndexHash {
    param([Parameter(Mandatory = $true)][string]$Path)

    return (Get-FileHash -LiteralPath (Join-Path $Path 'index.html') -Algorithm SHA256).Hash.ToUpperInvariant()
}

function Copy-DirectoryContents {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination
    )

    New-Item -ItemType Directory -Path $Destination -Force:$false | Out-Null
    foreach ($item in @(Get-ChildItem -LiteralPath $Source -Force)) {
        Copy-Item -LiteralPath $item.FullName -Destination (Join-Path $Destination $item.Name) -Recurse -Force
    }
}

function Write-Manifest {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][object]$Manifest
    )

    $temporaryPath = "$Path.pending"
    $Manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $temporaryPath -Encoding UTF8 -NoNewline
    # The destination is the manifest created by this same release path. The
    # release directory is rejected if it exists before Apply, so replacing
    # this file cannot overwrite an unrelated release record.
    Move-Item -LiteralPath $temporaryPath -Destination $Path -Force
}

if ([string]::IsNullOrWhiteSpace($StaticRoot)) {
    throw 'StaticRoot is required and must point to the directory containing the exact admin directory.'
}
$staticRootPath = Resolve-ExternalPath -Path $StaticRoot -Label 'StaticRoot'
$adminTargetPath = [System.IO.Path]::GetFullPath((Join-Path $staticRootPath 'admin'))
Assert-PathInside -Path $adminTargetPath -Root $staticRootPath -Label 'admin target'

if ($Action -eq 'Apply' -or $Action -eq 'VerifyRetention') {
    if ([string]::IsNullOrWhiteSpace($ArchiveRoot)) {
        throw "ArchiveRoot is required for $Action and must be outside StaticRoot."
    }
    $archiveRootPath = Resolve-ExternalPath -Path $ArchiveRoot -Label 'ArchiveRoot'
    $staticPrefix = $staticRootPath.TrimEnd([char[]]"/\") + [System.IO.Path]::DirectorySeparatorChar
    if ($archiveRootPath.StartsWith($staticPrefix, [System.StringComparison]::OrdinalIgnoreCase) -or
            $staticRootPath.StartsWith($archiveRootPath.TrimEnd([char[]]"/\") + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw 'ArchiveRoot must be separate from StaticRoot so archived files are not served by Caddy.'
    }
    if ($Action -eq 'Apply') {
        if (-not (Test-Path -LiteralPath $archiveRootPath -PathType Container)) {
            throw 'ArchiveRoot must already exist as a regular directory before Apply; the script will not create an unverified path.'
        }
        [void](Get-RequiredDirectory -Path $archiveRootPath -Label 'ArchiveRoot')
        Assert-NoReparsePoints -Root $archiveRootPath -Label 'ArchiveRoot'
    }
}

if ($Action -eq 'Plan' -or $Action -eq 'Apply') {
    if ([string]::IsNullOrWhiteSpace($ReleaseId)) {
        throw "$Action requires ReleaseId."
    }
    Assert-ReleaseId -Value $ReleaseId
    $sourcePath = Resolve-RepoPath -Path $SourceDist -Label 'SourceDist'
    [void](Get-RequiredDirectory -Path $sourcePath -Label 'SourceDist')
    $sourceIndex = Assert-AdminBuild -Path $sourcePath
    $sourceHash = Get-IndexHash -Path $sourcePath
    if ($Action -eq 'Apply') {
        if ([string]::IsNullOrWhiteSpace($ReleaseTicket)) {
            throw 'Apply requires ReleaseTicket for the release record.'
        }
        Assert-ReleaseTicket -Value $ReleaseTicket
        if ([string]::IsNullOrWhiteSpace($ExpectedIndexSha256)) {
            throw 'Apply requires ExpectedIndexSha256 from the reviewed build artifact.'
        }
        Assert-Sha256 -Value $ExpectedIndexSha256 -Label 'ExpectedIndexSha256'
        if ($sourceHash -ne $ExpectedIndexSha256.ToUpperInvariant()) {
            throw "Source index SHA-256 '$sourceHash' does not match ExpectedIndexSha256."
        }
        if (-not $AllowProductionSwitch -or -not $ConfirmProductionTarget) {
            throw 'Apply requires both -AllowProductionSwitch and -ConfirmProductionTarget.'
        }
    }
}

if ($Action -eq 'VerifyRetention') {
	[void](Get-RequiredDirectory -Path $archiveRootPath -Label 'ArchiveRoot')
	Assert-NoReparsePoints -Root $archiveRootPath -Label 'ArchiveRoot'
	$manifests = @(Get-ChildItem -LiteralPath $archiveRootPath -Filter 'manifest.json' -File -Recurse)
	if ($manifests.Count -eq 0) {
		throw "No release manifests found under ArchiveRoot: $archiveRootPath"
	}
	$now = [DateTimeOffset]::UtcNow
	$checks = @()
	foreach ($manifestFile in $manifests) {
		[void](Get-RequiredFile -Path $manifestFile.FullName -Label 'release manifest')
		$releaseDirectory = (Get-Item -LiteralPath $manifestFile.FullName).Directory
		if ($null -eq $releaseDirectory -or $null -eq $releaseDirectory.Parent -or
				-not $releaseDirectory.Parent.FullName.Equals($archiveRootPath, [StringComparison]::OrdinalIgnoreCase)) {
			throw "Release manifest must be directly under ArchiveRoot/<releaseId>: $($manifestFile.FullName)"
		}
		try {
			$manifest = Get-Content -LiteralPath $manifestFile.FullName -Raw | ConvertFrom-Json
		} catch {
			throw "Release manifest is not valid JSON: $($manifestFile.FullName)"
		}
		if ([string]$manifest.status -ne 'active' -or
				[string]::IsNullOrWhiteSpace([string]$manifest.retentionUntilUtc) -or
				[string]::IsNullOrWhiteSpace([string]$manifest.previousIndexSha256)) {
			throw "Release manifest is incomplete or not active: $($manifestFile.FullName)"
		}
		$manifestReleaseId = [string]$manifest.releaseId
		Assert-ReleaseId -Value $manifestReleaseId
		if ($manifestFile.Directory.Name -ne $manifestReleaseId) {
			throw "Release manifest directory does not match releaseId: $($manifestFile.FullName)"
		}
		$manifestReleaseTicket = [string]$manifest.releaseTicket
		Assert-ReleaseTicket -Value $manifestReleaseTicket
		Assert-Sha256 -Value ([string]$manifest.previousIndexSha256) -Label 'previousIndexSha256'
		try {
			$retentionUntil = [DateTimeOffset]::Parse([string]$manifest.retentionUntilUtc).ToUniversalTime()
		} catch {
			throw "Release manifest retentionUntilUtc is not a valid timestamp: $($manifestFile.FullName)"
		}
		$archivedAdmin = Join-Path $manifestFile.Directory.FullName 'admin'
		$archivedAdminExists = Test-Path -LiteralPath $archivedAdmin -PathType Container
		if ($archivedAdminExists) {
			[void](Get-RequiredDirectory -Path $archivedAdmin -Label 'archived admin artifact')
			Assert-NoReparsePoints -Root $archivedAdmin -Label 'archived admin artifact'
		}
		$archivedIndex = Join-Path $archivedAdmin 'index.html'
		$artifactExists = $archivedAdminExists -and (Test-Path -LiteralPath $archivedIndex -PathType Leaf)
		$hashMatches = $false
		if ($artifactExists) {
			$hashMatches = (Get-IndexHash -Path $archivedAdmin) -eq ([string]$manifest.previousIndexSha256).ToUpperInvariant()
		}
		$checks += [ordered]@{
			releaseId = $manifestReleaseId
			releaseTicket = $manifestReleaseTicket
			retentionUntilUtc = $retentionUntil.ToString('o')
            artifactExists = $artifactExists
            previousHashMatches = $hashMatches
            retentionSatisfied = $now -ge $retentionUntil -and $artifactExists -and $hashMatches
        }
    }
    $report = [ordered]@{
        action = 'VerifyRetention'
        archiveRoot = $archiveRootPath
        checkedAtUtc = $now.ToString('o')
        releases = @($checks)
        pass = @($checks | Where-Object { -not $_.retentionSatisfied }).Count -eq 0
        sideEffects = 'read-only'
    }
    $report | ConvertTo-Json -Depth 8
    if (-not $report.pass) {
        exit 2
    }
    exit 0
}

[void](Get-RequiredDirectory -Path $staticRootPath -Label 'StaticRoot')
[void](Assert-NoReparsePoints -Root $staticRootPath -Label 'StaticRoot')
[void](Get-RequiredDirectory -Path $adminTargetPath -Label 'current admin static directory')
$currentIndex = Get-RequiredFile -Path (Join-Path $adminTargetPath 'index.html') -Label 'current admin index.html'
Assert-NoReparsePoints -Root $adminTargetPath -Label 'current admin artifact'
$currentHash = Get-IndexHash -Path $adminTargetPath
if ($currentHash -eq $sourceHash) {
    throw 'Source admin-next artifact has the same index hash as the current admin artifact; refusing a no-op release.'
}

$retentionUntil = [DateTimeOffset]::UtcNow.AddHours($RetentionHours)
$preview = [ordered]@{
    action = $Action
    staticRoot = $staticRootPath
    target = $adminTargetPath
    source = $sourcePath
    currentIndexSha256 = $currentHash
    newIndexSha256 = $sourceHash
    releaseId = $ReleaseId
    retentionHours = $RetentionHours
    retentionUntilUtc = $retentionUntil.ToString('o')
}

if ($Action -eq 'Plan') {
    $preview.sideEffects = 'read-only'
    $preview | ConvertTo-Json -Depth 8
    exit 0
}

$releasePath = Join-Path $archiveRootPath $ReleaseId
$archivedAdminPath = Join-Path $releasePath 'admin'
$manifestPath = Join-Path $releasePath 'manifest.json'
$stagingPath = Join-Path $staticRootPath ".admin-staging-$ReleaseId"
$failedPath = Join-Path $releasePath 'failed-admin'
foreach ($path in @($releasePath, $stagingPath, $failedPath)) {
    if (Test-Path -LiteralPath $path) {
        throw "Refusing to overwrite an existing release or staging path: $path"
    }
}

New-Item -ItemType Directory -Path $releasePath -Force:$false | Out-Null
$manifest = [ordered]@{
    schemaVersion = 1
    status = 'prepared'
    releaseId = $ReleaseId
    releaseTicket = $ReleaseTicket
    switchedAtUtc = $null
    retentionUntilUtc = $retentionUntil.ToString('o')
    sourceDistribution = (Get-RelativePath -Path $sourcePath -Root $repoRoot)
    currentTarget = 'admin'
    previousIndexSha256 = $currentHash
    newIndexSha256 = $sourceHash
    sideEffects = 'production static directory replacement only; no Caddy reload'
}

$oldMoved = $false
$newMoved = $false
try {
    Copy-DirectoryContents -Source $sourcePath -Destination $stagingPath
    [void](Assert-AdminBuild -Path $stagingPath)
    if ((Get-IndexHash -Path $stagingPath) -ne $ExpectedIndexSha256.ToUpperInvariant()) {
        throw 'Staged admin-next artifact hash changed during copy.'
    }
    Write-Manifest -Path $manifestPath -Manifest $manifest

    Move-Item -LiteralPath $adminTargetPath -Destination $archivedAdminPath -Force:$false
    $oldMoved = $true
    Move-Item -LiteralPath $stagingPath -Destination $adminTargetPath -Force:$false
    $newMoved = $true
    if ((Get-IndexHash -Path $adminTargetPath) -ne $ExpectedIndexSha256.ToUpperInvariant()) {
        throw 'Live admin artifact hash does not match the reviewed build artifact.'
    }

    $manifest.status = 'active'
    $manifest.switchedAtUtc = [DateTimeOffset]::UtcNow.ToString('o')
    Write-Manifest -Path $manifestPath -Manifest $manifest
    $preview.status = 'active'
    $preview.archive = $archivedAdminPath
    $preview.manifest = $manifestPath
    $preview.sideEffects = 'admin static directory replaced; old artifact archived; no Caddy reload'
    $preview | ConvertTo-Json -Depth 8
}
catch {
    $failure = $_
    try {
        if ($newMoved -and (Test-Path -LiteralPath $adminTargetPath)) {
            Move-Item -LiteralPath $adminTargetPath -Destination $failedPath -Force:$false
        }
        if ($oldMoved -and -not (Test-Path -LiteralPath $adminTargetPath) -and (Test-Path -LiteralPath $archivedAdminPath)) {
            Move-Item -LiteralPath $archivedAdminPath -Destination $adminTargetPath -Force:$false
        }
    }
    catch {
        throw "Static switch failed and automatic restoration also failed. Original=$($failure.Exception.Message); restore=$($_.Exception.Message)"
    }
    throw "Static switch failed; old artifact was restored. $($failure.Exception.Message)"
}
