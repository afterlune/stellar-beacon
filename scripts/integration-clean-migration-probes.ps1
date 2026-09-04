[CmdletBinding()]
param(
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$LASTEXITCODE = 0

if (-not $AllowWrites) {
    throw 'Refusing to remove isolated migration probe schemas. Re-run with -AllowWrites during an approved benetnasch-integration repair window.'
}

# Ignore local Compose overrides. This repair is valid only for the fixed
# integration project and the explicitly named test-artifact schemas below.
$env:INTEGRATION_USE_BASE_COMPOSE = '1'
. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$composeArgs = @(Get-IntegrationComposeArgs)

function Invoke-DockerCapture {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $output = @(& docker @Arguments 2>&1 | ForEach-Object { [string]$_ })
    $exitCode = [int]$LASTEXITCODE
    if ($exitCode -ne 0) {
        throw "docker command failed with exit code $exitCode"
    }
    return $output
}

function Invoke-IntegrationPsql {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Sql
    )

    $lines = Invoke-DockerCapture -Arguments ($composeArgs + @(
        'exec', '-T', 'postgresql', 'psql',
        '-X', '-q', '-U', 'postgres', '-d', 'benetnasch',
        '-v', 'ON_ERROR_STOP=1', '-At', '-F', '|', '-c', $Sql
    ))
    return (($lines -join "`n").Trim())
}

$services = @(Invoke-DockerCapture -Arguments ($composeArgs + @('config', '--services')))
if ('postgresql' -notin $services) {
    throw 'The selected Compose project does not expose the isolated postgresql service.'
}

$statusLines = @(Invoke-DockerCapture -Arguments ($composeArgs + @('ps', '--format', 'json')))
$records = @()
foreach ($line in $statusLines) {
    if ([string]::IsNullOrWhiteSpace($line)) {
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
    throw 'The isolated postgresql container is not running; no probe schema was changed.'
}
if ([string]$postgres[0].Name -notlike 'benetnasch-integration-*') {
    throw 'The selected postgresql container is outside the benetnasch-integration project.'
}

$expectedRelations = @(
    'migration_failure_probe',
    'migration_failure_probe_pkey',
    'migration_probe',
    'migration_probe_pkey',
    'schema_migrations',
    'schema_migrations_pkey'
) | Sort-Object
$probeQuery = @'
SELECT n.nspname,
       COALESCE(string_agg(c.relname, ',' ORDER BY c.relname), '')
FROM pg_namespace AS n
LEFT JOIN pg_class AS c ON c.relnamespace = n.oid
WHERE n.nspname ~ '^migration_probe_[0-9]+$'
GROUP BY n.nspname
ORDER BY n.nspname;
'@
$probeOutput = Invoke-IntegrationPsql -Sql $probeQuery
$probeSchemas = [System.Collections.Generic.List[string]]::new()
foreach ($line in ($probeOutput -split "`r?`n")) {
    if ([string]::IsNullOrWhiteSpace($line)) {
        continue
    }
    $parts = $line -split '\|', 2
    if ($parts.Count -ne 2) {
        throw 'The migration probe inventory had an invalid row; no schema was changed.'
    }
    $schema = [string]$parts[0]
    if ($schema -notmatch '^migration_probe_[0-9]+$') {
        throw "Refusing unexpected migration probe schema name '$schema'."
    }
    $actualRelations = @()
    if (-not [string]::IsNullOrWhiteSpace([string]$parts[1])) {
        $actualRelations = @($parts[1] -split ',' | Sort-Object)
    }
    if (($actualRelations -join '|') -ne ($expectedRelations -join '|')) {
        throw "Refusing schema '$schema' because its relation set is not an exact migration-test artifact."
    }
    $probeSchemas.Add($schema)
}

if ($probeSchemas.Count -eq 0) {
    Write-Host 'No stale isolated migration probe schemas found.' -ForegroundColor Green
    return
}

Write-Host "Removing $($probeSchemas.Count) verified isolated migration probe schema(s): $($probeSchemas -join ', ')" -ForegroundColor Yellow
foreach ($schema in $probeSchemas) {
    # The name was checked against a numeric-only allowlist above. Quote it as
    # an identifier as a second defense before passing it to psql.
    $quotedSchema = '"' + $schema.Replace('"', '""') + '"'
    $dropSQL = "DROP SCHEMA IF EXISTS $quotedSchema CASCADE;"
    Invoke-DockerCapture -Arguments ($composeArgs + @(
        'exec', '-T', 'postgresql', 'psql',
        '-X', '-q', '-U', 'postgres', '-d', 'benetnasch',
        '-v', 'ON_ERROR_STOP=1', '-c', $dropSQL
    )) | Out-Null
}

$remaining = Invoke-IntegrationPsql -Sql "SELECT count(*) FROM pg_namespace WHERE nspname ~ '^migration_probe_[0-9]+$';"
if ([int]$remaining -ne 0) {
    throw "Migration probe cleanup did not converge; $remaining verified probe schema(s) remain."
}

Write-Host 'Verified isolated migration probe schemas were removed; no production target was used.' -ForegroundColor Green
