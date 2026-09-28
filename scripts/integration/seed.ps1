. (Join-Path $PSScriptRoot 'common.ps1')
Import-IntegrationEnv

Wait-IntegrationHttp -Uri 'http://127.0.0.1:17700/health'
Invoke-IntegrationCompose -Arguments @('exec', '-T', 'postgresql', 'pg_isready', '-U', 'postgres', '-d', 'stellar_beacon')

# Keep repeated integration runs idempotent when the named volume already
# contains data from an earlier run.
& (Join-Path $PSScriptRoot 'repair-sequences.ps1')
if ($LASTEXITCODE -ne 0) {
    throw "integration sequence repair failed with exit code $LASTEXITCODE"
}

Push-Location $script:IntegrationRepoRoot
try {
    & go run ./cmd/integration-seed
    if ($LASTEXITCODE -ne 0) {
        throw "integration database seed failed with exit code $LASTEXITCODE"
    }
} finally {
    Pop-Location
}

# Website configuration changes must bypass the process-external cache so the
# next API read observes the deterministic review/email settings written above.
Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'redis', 'redis-cli', '--no-auth-warning', '-a', $env:REDIS_PASSWORD,
    '-n', '1', 'DEL', 'website_config'
) | Out-Null

# Comment notification emails carry a 24h per-recipient budget in Redis. A
# repeated verification run must start from an empty budget, otherwise the
# notification assertions silently starve and fail for the wrong reason.
$redisCli = @('exec', '-T', 'redis', 'redis-cli', '--no-auth-warning', '-a', $env:REDIS_PASSWORD, '-n', '1')
$notificationKeys = @(& docker @((Get-IntegrationComposeArgs) + $redisCli + @('--scan', '--pattern', 'comment-notify*')) 2>$null)
$notificationKeys = @($notificationKeys | Where-Object { $_ -and $_.Trim() -ne '' })
if ($notificationKeys.Count -gt 0) {
    Write-Host "Clearing $($notificationKeys.Count) stale comment notification limiters..." -ForegroundColor Cyan
    Invoke-IntegrationCompose -Arguments ($redisCli + @('DEL') + $notificationKeys) | Out-Null
}

Invoke-IntegrationCompose -Arguments @(
    'exec', '-T', 'backend', '/app/stellar-beacon', 'search-reindex'
)

Write-Host 'Integration database users and Meilisearch index are ready.' -ForegroundColor Green
