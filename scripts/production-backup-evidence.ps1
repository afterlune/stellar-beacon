[CmdletBinding()]
param(
    [ValidateSet('Plan', 'Verify')]
    [string]$Action = 'Plan',
    [string]$BackupDirectory,
    [string]$ReleaseTicket,
    [ValidateRange(1, 8760)]
    [int]$MinimumRetentionHours = 168
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repoRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot)).TrimEnd(
    [IO.Path]::DirectorySeparatorChar,
    [IO.Path]::AltDirectorySeparatorChar
)

function Get-MetadataValue {
    param(
        [Parameter(Mandatory = $true)]$Metadata,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $property = $Metadata.PSObject.Properties[$Name]
    if ($null -eq $property) {
        return $null
    }
    return $property.Value
}

function Get-RequiredBoolean {
    param(
        [Parameter(Mandatory = $true)]$Metadata,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $value = Get-MetadataValue -Metadata $Metadata -Name $Name
    if ($value -isnot [bool]) {
        throw "Backup metadata field '$Name' must be a JSON boolean."
    }
    return [bool]$value
}

function Assert-SafeBackupDirectory {
    param([Parameter(Mandatory = $true)][string]$RequestedPath)

    if ([string]::IsNullOrWhiteSpace($RequestedPath)) {
        throw 'Verify requires -BackupDirectory.'
    }

    try {
        $fullPath = [IO.Path]::GetFullPath($RequestedPath).TrimEnd(
            [IO.Path]::DirectorySeparatorChar,
            [IO.Path]::AltDirectorySeparatorChar
        )
    } catch {
        throw 'BackupDirectory is not a valid path.'
    }

    if ($fullPath.Equals($repoRoot, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'BackupDirectory must be outside the repository.'
    }

    $repoPrefix = $repoRoot + [IO.Path]::DirectorySeparatorChar
    if ($fullPath.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'BackupDirectory must be outside the repository.'
    }

    $root = [IO.Path]::GetPathRoot($fullPath).TrimEnd(
        [IO.Path]::DirectorySeparatorChar,
        [IO.Path]::AltDirectorySeparatorChar
    )
    if ($fullPath.Equals($root, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'BackupDirectory cannot be a filesystem root.'
    }

    if (-not (Test-Path -LiteralPath $fullPath -PathType Container)) {
        throw 'BackupDirectory does not exist.'
    }

    $directory = Get-Item -LiteralPath $fullPath
    if (($directory.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw 'BackupDirectory must not be a reparse point or symlink.'
    }

    return $fullPath
}

function Get-RequiredArtifact {
    param(
        [Parameter(Mandatory = $true)][string]$Directory,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $path = Join-Path $Directory $Name
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required backup artifact is missing: $Name"
    }

    $item = Get-Item -LiteralPath $path
    if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "Backup artifact must not be a reparse point: $Name"
    }
    if ($item.Length -le 0) {
        throw "Backup artifact is empty: $Name"
    }
    return $path
}

function Parse-UtcTimestamp {
    param(
        [Parameter(Mandatory = $true)]$Value,
        [Parameter(Mandatory = $true)][string]$FieldName
    )

    if ($null -eq $Value -or [string]::IsNullOrWhiteSpace([string]$Value)) {
        throw "Backup metadata field '$FieldName' is missing."
    }
    try {
        return [DateTimeOffset]::Parse(
            [string]$Value,
            [Globalization.CultureInfo]::InvariantCulture,
            [Globalization.DateTimeStyles]::AssumeUniversal -bor [Globalization.DateTimeStyles]::AdjustToUniversal
        )
    } catch {
        throw "Backup metadata field '$FieldName' is not a valid timestamp."
    }
}

function Assert-Sha256 {
    param(
        [Parameter(Mandatory = $true)][string]$Value,
        [Parameter(Mandatory = $true)][string]$FieldName
    )

    if ($Value -notmatch '^[0-9a-fA-F]{64}$') {
        throw "Backup metadata field '$FieldName' is not a SHA-256 digest."
    }
}

function Assert-ProductionBackupEvidence {
    param(
        [Parameter(Mandatory = $true)][string]$Directory,
        [Parameter(Mandatory = $true)][string]$Ticket,
        [Parameter(Mandatory = $true)][int]$RetentionHours
    )

    if ([string]::IsNullOrWhiteSpace($Ticket)) {
        throw 'Verify requires -ReleaseTicket.'
    }

    $dumpPath = Get-RequiredArtifact -Directory $Directory -Name 'benetnasch.dump'
    $globalsPath = Get-RequiredArtifact -Directory $Directory -Name 'postgres-globals.sql'
    $metadataPath = Get-RequiredArtifact -Directory $Directory -Name 'metadata.json'

    try {
        $metadata = Get-Content -LiteralPath $metadataPath -Raw | ConvertFrom-Json
    } catch {
        throw 'Backup metadata.json is not valid JSON.'
    }

    if ([string](Get-MetadataValue -Metadata $metadata -Name 'format') -ne 'benetnasch.production-backup.v1') {
        throw 'Backup metadata format is not benetnasch.production-backup.v1.'
    }
    if ([string](Get-MetadataValue -Metadata $metadata -Name 'environment') -ne 'production') {
        throw 'Backup metadata environment is not production.'
    }
    if ([string](Get-MetadataValue -Metadata $metadata -Name 'releaseTicket') -ne $Ticket) {
        throw 'Backup metadata does not match the supplied release ticket.'
    }
    if ([string](Get-MetadataValue -Metadata $metadata -Name 'sourceDatabase') -ne 'benetnasch') {
        throw 'Backup metadata source database is not benetnasch.'
    }
    if ([string](Get-MetadataValue -Metadata $metadata -Name 'dumpFile') -ne 'benetnasch.dump' -or
        [string](Get-MetadataValue -Metadata $metadata -Name 'globalsFile') -ne 'postgres-globals.sql') {
        throw 'Backup metadata names do not match the required artifact files.'
    }
    if (-not (Get-RequiredBoolean -Metadata $metadata -Name 'restoreVerified')) {
        throw 'Backup metadata does not confirm a successful isolated restore verification.'
    }
    if ([string](Get-MetadataValue -Metadata $metadata -Name 'restoreEnvironment') -ne 'isolated') {
        throw 'Backup metadata restore environment is not isolated.'
    }

    $generatedAt = Parse-UtcTimestamp -Value (Get-MetadataValue -Metadata $metadata -Name 'generatedAtUtc') -FieldName 'generatedAtUtc'
    $restoreVerifiedAt = Parse-UtcTimestamp -Value (Get-MetadataValue -Metadata $metadata -Name 'restoreVerifiedAtUtc') -FieldName 'restoreVerifiedAtUtc'
    $retentionUntil = Parse-UtcTimestamp -Value (Get-MetadataValue -Metadata $metadata -Name 'retentionUntilUtc') -FieldName 'retentionUntilUtc'
    $now = [DateTimeOffset]::UtcNow

    if ($generatedAt -gt $now.AddMinutes(5)) {
        throw 'Backup metadata generatedAtUtc is unexpectedly in the future.'
    }
    if ($restoreVerifiedAt -gt $now.AddMinutes(5)) {
        throw 'Backup metadata restoreVerifiedAtUtc is unexpectedly in the future.'
    }
    if ($restoreVerifiedAt -lt $generatedAt) {
        throw 'Restore verification predates backup generation.'
    }
    if ($retentionUntil -lt $generatedAt.AddHours($RetentionHours)) {
        throw 'Backup retention period is shorter than the release requirement.'
    }
    if ($retentionUntil -le $now) {
        throw 'Backup retention period has already expired.'
    }

    $dumpDigest = [string](Get-MetadataValue -Metadata $metadata -Name 'dumpSha256')
    $globalsDigest = [string](Get-MetadataValue -Metadata $metadata -Name 'globalsSha256')
    Assert-Sha256 -Value $dumpDigest -FieldName 'dumpSha256'
    Assert-Sha256 -Value $globalsDigest -FieldName 'globalsSha256'

    $actualDumpDigest = (Get-FileHash -Algorithm SHA256 -LiteralPath $dumpPath).Hash
    $actualGlobalsDigest = (Get-FileHash -Algorithm SHA256 -LiteralPath $globalsPath).Hash
    if (-not $actualDumpDigest.Equals($dumpDigest, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Custom dump SHA-256 does not match backup metadata.'
    }
    if (-not $actualGlobalsDigest.Equals($globalsDigest, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Globals SQL SHA-256 does not match backup metadata.'
    }

    Write-Host 'Production PostgreSQL backup evidence passed (read-only verification).' -ForegroundColor Green
    Write-Host ("Artifacts are retained until {0}." -f $retentionUntil.ToUniversalTime().ToString('o'))
}

if ($Action -eq 'Plan') {
    Write-Host 'Production backup evidence plan (read-only):' -ForegroundColor Cyan
    Write-Host '- the deployment platform creates the backup and performs an isolated restore;'
    Write-Host '- metadata.json records the release ticket, source database, SHA-256 digests, restore result, and retention deadline;'
    Write-Host '- this repository script only verifies those artifacts and never connects to PostgreSQL, Docker, or a production service;'
    Write-Host 'Run Verify only after the approved release window has attached the platform artifacts.' -ForegroundColor Yellow
    exit 0
}

$safeDirectory = Assert-SafeBackupDirectory -RequestedPath $BackupDirectory
Assert-ProductionBackupEvidence -Directory $safeDirectory -Ticket $ReleaseTicket -RetentionHours $MinimumRetentionHours
