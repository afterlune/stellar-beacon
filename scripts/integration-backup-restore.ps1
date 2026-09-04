[CmdletBinding()]
param(
    [switch]$AllowWrites,
    [switch]$AllowRestore,
    [string]$BackupDirectory,
    [switch]$KeepArtifacts
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($AllowRestore -and -not $AllowWrites) {
    throw 'The -AllowRestore permission requires -AllowWrites; backup artifacts must be explicitly authorized too.'
}
if (-not $AllowWrites) {
    throw 'Refusing the isolated backup export. Re-run with -AllowWrites during an approved benetnasch-integration window.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')

# Always resolve the source Compose file and the fixed integration project. A
# local override is useful for ordinary development, but it is too broad for
# a backup/restore operation because it could redirect a volume to production.
$env:INTEGRATION_USE_BASE_COMPOSE = '1'
Import-IntegrationEnv

function Invoke-IntegrationComposeCapture {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $dockerArgs = @(Get-IntegrationComposeArgs) + $Arguments
    $output = & docker @dockerArgs
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        throw "docker compose failed with exit code $exitCode"
    }
    return @($output | ForEach-Object { [string]$_ })
}

function Invoke-IntegrationPsql {
    param(
        [Parameter(Mandatory = $true)][string]$Database,
        [Parameter(Mandatory = $true)][string]$Sql
    )

    $lines = Invoke-IntegrationComposeCapture -Arguments @(
        'exec', '-T', 'postgresql', 'psql',
        '-X', '-q', '-U', 'postgres', '-d', $Database,
        '-v', 'ON_ERROR_STOP=1', '-At', '-c', $Sql
    )
    return (($lines -join "`n").Trim())
}

function Assert-IntegrationContainers {
    $composeArgs = @(Get-IntegrationComposeArgs)
    $services = @(& docker @($composeArgs + @('config', '--services')))
    if ($LASTEXITCODE -ne 0 -or 'postgresql' -notin $services) {
        throw 'The selected Compose project does not expose the isolated postgresql service.'
    }

    $psOutput = @(& docker @($composeArgs + @('ps', '--format', 'json')))
    if ($LASTEXITCODE -ne 0) {
        throw 'Unable to inspect the isolated Compose project.'
    }
    $records = @()
    foreach ($line in $psOutput) {
        if ([string]::IsNullOrWhiteSpace([string]$line)) {
            continue
        }
        try {
            $records += [string]$line | ConvertFrom-Json
        } catch {
            throw 'The isolated Compose status output was not valid JSON.'
        }
    }
    $postgres = @($records | Where-Object { [string]$_.Service -eq 'postgresql' })
    if ($postgres.Count -ne 1 -or [string]$postgres[0].State -notin @('running', 'Up')) {
        throw 'The isolated postgresql container is not running; no backup or restore was attempted.'
    }
    if ([string]$postgres[0].Name -notlike 'benetnasch-integration-*') {
        throw 'The selected postgresql container is outside the benetnasch-integration project.'
    }
}

function Get-DatabaseSnapshot {
    param([Parameter(Mandatory = $true)][string]$Database)

    $sql = @'
SELECT json_build_object(
    'serverVersion', current_setting('server_version'),
    'schemaMigrations', (SELECT count(*) FROM public.schema_migrations),
    'migrationDigest', md5(COALESCE((SELECT string_agg(version::text || ':' || name || ':' || checksum, '|' ORDER BY version) FROM public.schema_migrations), '')),
    'articles', (SELECT count(*) FROM public.t_article),
    'users', (SELECT count(*) FROM public.t_user_info),
    'authUsers', (SELECT count(*) FROM public.t_user_auth),
    'comments', (SELECT count(*) FROM public.t_comment),
    'operationLogs', (SELECT count(*) FROM public.t_operation_log),
    'aiJobs', (SELECT count(*) FROM public.t_ai_job),
    'aiReviews', (SELECT count(*) FROM public.t_ai_review),
    'agentTaskRuns', (SELECT count(*) FROM public.t_agent_task_run),
    'agentProfiles', (SELECT count(*) FROM public.t_agent_profile),
    'contentProjections', (SELECT count(*) FROM public.t_content_projection)
)::text;
'@
    $raw = Invoke-IntegrationPsql -Database $Database -Sql $sql
    if ([string]::IsNullOrWhiteSpace($raw)) {
        throw "Database snapshot for '$Database' was empty."
    }
    try {
        return $raw | ConvertFrom-Json
    } catch {
        throw "Database snapshot for '$Database' was not valid JSON."
    }
}

function Assert-SnapshotMatches {
    param(
        [Parameter(Mandatory = $true)]$Expected,
        [Parameter(Mandatory = $true)]$Actual
    )

    $properties = @(
        'schemaMigrations', 'migrationDigest', 'articles', 'users', 'authUsers',
        'comments', 'operationLogs', 'aiJobs', 'aiReviews', 'agentTaskRuns',
        'agentProfiles', 'contentProjections'
    )
    foreach ($property in $properties) {
        if ([string]$Expected.$property -ne [string]$Actual.$property) {
            throw "Restored database verification failed for '$property'."
        }
    }
}

function Get-SafeBackupDirectory {
    param([string]$Requested)

    $repoRoot = [IO.Path]::GetFullPath($script:IntegrationRepoRoot).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar)
    if ([string]::IsNullOrWhiteSpace($Requested)) {
        $base = [IO.Path]::GetTempPath()
        return Join-Path $base ("benetnasch-integration-backup-{0}" -f (Get-Date).ToUniversalTime().ToString('yyyyMMddTHHmmssfffZ'))
    }

    $full = [IO.Path]::GetFullPath($Requested)
    $repoPrefix = $repoRoot + [IO.Path]::DirectorySeparatorChar
    if ($full.Equals($repoRoot, [StringComparison]::OrdinalIgnoreCase) -or
        $full.StartsWith($repoPrefix, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'BackupDirectory must be outside the repository so database exports cannot enter Git.'
    }
    return $full
}

Assert-IntegrationContainers
$outputDirectory = Get-SafeBackupDirectory -Requested $BackupDirectory
New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null

$dumpPath = Join-Path $outputDirectory 'benetnasch.dump'
$globalsPath = Join-Path $outputDirectory 'postgres-globals.sql'
$metadataPath = Join-Path $outputDirectory 'metadata.json'
foreach ($path in @($dumpPath, $globalsPath, $metadataPath)) {
    if (Test-Path -LiteralPath $path) {
        throw "Refusing to overwrite an existing backup artifact: $path"
    }
}

$containerToken = [Guid]::NewGuid().ToString('N')
$containerDumpPath = "/tmp/benetnasch-$containerToken.dump"
$containerGlobalsPath = "/tmp/benetnasch-$containerToken-globals.sql"
$restoreDatabase = $null
$sourceSnapshot = Get-DatabaseSnapshot -Database 'benetnasch'
$backupStarted = (Get-Date).ToUniversalTime()

try {
    Invoke-IntegrationCompose -Arguments @(
        'exec', '-T', 'postgresql', 'pg_dump',
        '--format=custom', '--no-owner', '--no-privileges',
        '--username=postgres', '--dbname=benetnasch',
        '--file', $containerDumpPath
    )
    Invoke-IntegrationCompose -Arguments @(
        'exec', '-T', 'postgresql', 'pg_dumpall',
        '--globals-only', '--no-role-passwords',
        '--username=postgres', '--database=postgres',
        '--file', $containerGlobalsPath
    )
    Invoke-IntegrationCompose -Arguments @('cp', "postgresql:$containerDumpPath", $dumpPath)
    Invoke-IntegrationCompose -Arguments @('cp', "postgresql:$containerGlobalsPath", $globalsPath)
} finally {
    try {
        Invoke-IntegrationCompose -Arguments @(
            'exec', '-T', 'postgresql', 'rm', '-f',
            $containerDumpPath, $containerGlobalsPath
        )
    } catch {
        Write-Warning 'Unable to remove temporary backup files from the isolated PostgreSQL container.'
    }
}

foreach ($path in @($dumpPath, $globalsPath)) {
    if (-not (Test-Path -LiteralPath $path) -or (Get-Item -LiteralPath $path).Length -le 0) {
        throw "Backup artifact is missing or empty: $path"
    }
}

$restoreVerified = $false
if ($AllowRestore) {
    # Copy the custom dump back into the isolated container for a clean-database
    # restore. The target database is unique and is never the integration source.
    $containerRestoreDumpPath = "/tmp/benetnasch-$containerToken-restore.dump"
    $restoreDatabase = "benetnasch_restore_$($containerToken.Substring(0, 12))"
    $restoreCreated = $false
    try {
        Invoke-IntegrationCompose -Arguments @('cp', $dumpPath, "postgresql:$containerRestoreDumpPath")
        Invoke-IntegrationCompose -Arguments @('exec', '-T', 'postgresql', 'createdb', '-U', 'postgres', $restoreDatabase)
        $restoreCreated = $true
        Invoke-IntegrationCompose -Arguments @(
            'exec', '-T', 'postgresql', 'pg_restore',
            '--exit-on-error', '--no-owner', '--no-privileges',
            '--username=postgres', '--dbname', $restoreDatabase,
            $containerRestoreDumpPath
        )
        $restoredSnapshot = Get-DatabaseSnapshot -Database $restoreDatabase
        Assert-SnapshotMatches -Expected $sourceSnapshot -Actual $restoredSnapshot
        $restoreVerified = $true
    } finally {
        try {
            Invoke-IntegrationCompose -Arguments @(
                'exec', '-T', 'postgresql', 'rm', '-f', $containerRestoreDumpPath
            )
        } catch {
            Write-Warning 'Unable to remove temporary restore dump from isolated PostgreSQL container.'
        }
        if ($restoreCreated -and -not $KeepArtifacts) {
            try {
                Invoke-IntegrationCompose -Arguments @('exec', '-T', 'postgresql', 'dropdb', '-U', 'postgres', '--if-exists', $restoreDatabase)
            } catch {
                Write-Warning "Unable to remove temporary restore database '$restoreDatabase'."
            }
        }
    }
}

$metadata = [ordered]@{
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString('o')
    backupStartedAtUtc = $backupStarted.ToString('o')
    composeProject = 'benetnasch-integration'
    sourceDatabase = 'benetnasch'
    restoreDatabase = $restoreDatabase
    serverVersion = [string]$sourceSnapshot.serverVersion
    sourceSnapshot = $sourceSnapshot
    dumpSha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $dumpPath).Hash.ToLowerInvariant()
    globalsSha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $globalsPath).Hash.ToLowerInvariant()
    restoreVerified = $restoreVerified
    artifactsKept = $true
}
$metadata | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $metadataPath -Encoding UTF8 -NoNewline

if ($AllowRestore) {
    Write-Host 'Isolated PostgreSQL backup and clean-database restore verification passed.' -ForegroundColor Green
} else {
    Write-Host 'Isolated PostgreSQL backup export passed; restore verification was not requested.' -ForegroundColor Green
}
Write-Host "Backup artifacts: $outputDirectory"
if ($AllowRestore -and -not $KeepArtifacts) {
    Write-Host 'The temporary restore database was removed; backup artifacts remain for the release record.' -ForegroundColor Yellow
}
