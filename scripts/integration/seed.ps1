. (Join-Path $PSScriptRoot 'common.ps1')
Import-IntegrationEnv

Wait-IntegrationHttp -Uri 'http://127.0.0.1:17700/health'
Invoke-IntegrationCompose -Arguments @('exec', '-T', 'postgresql', 'pg_isready', '-U', 'postgres', '-d', 'stellar_beacon')

# The dump contains historical identity sequence values.  Keep reseeding
# idempotent when the integration volume already exists, too.
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

$meiliBase = 'http://127.0.0.1:17700'
$meiliHeaders = @{ Authorization = "Bearer $env:MEILI_MASTER_KEY" }
$indexBody = '{"uid":"articles","primaryKey":"id"}'
try {
    $indexTask = Invoke-RestMethod -Method Post -Uri "$meiliBase/indexes" -Headers $meiliHeaders -ContentType 'application/json' -Body $indexBody
    if ($null -ne $indexTask.taskUid) {
        Wait-IntegrationMeiliTask -TaskUid ([string]$indexTask.taskUid)
    }
} catch {
    $status = 0
    if ($_.Exception.Response) { $status = [int]$_.Exception.Response.StatusCode }
    if ($status -ne 409 -and $_.Exception.Message -notmatch 'index_already_exists') { throw }
}

$documents = @(
    @{ id = 147; articleTitle = 'Integration smoke article'; articleContent = 'stellar-beacon integration search'; isDelete = 0; status = 1; moderationStatus = 'visible' },
    @{ id = 152; articleTitle = 'Caddy integration article'; articleContent = 'frontend backend caddy minio'; isDelete = 0; status = 1; moderationStatus = 'visible' }
)
$documentsBody = $documents | ConvertTo-Json -Depth 5 -Compress
$task = Invoke-RestMethod -Method Post -Uri "$meiliBase/indexes/articles/documents" -Headers $meiliHeaders -ContentType 'application/json' -Body $documentsBody

if ($null -ne $task.taskUid) {
    Wait-IntegrationMeiliTask -TaskUid ([string]$task.taskUid)
}

$settingsBody = '["articleTitle","articleContent"]'
$settingsTask = Invoke-RestMethod -Method Put -Uri "$meiliBase/indexes/articles/settings/searchable-attributes" -Headers $meiliHeaders -ContentType 'application/json' -Body $settingsBody
if ($null -ne $settingsTask.taskUid) {
    Wait-IntegrationMeiliTask -TaskUid ([string]$settingsTask.taskUid)
}

$filterableBody = '["isDelete","status","moderationStatus"]'
$filterableTask = Invoke-RestMethod -Method Put -Uri "$meiliBase/indexes/articles/settings/filterable-attributes" -Headers $meiliHeaders -ContentType 'application/json' -Body $filterableBody
if ($null -ne $filterableTask.taskUid) {
    Wait-IntegrationMeiliTask -TaskUid ([string]$filterableTask.taskUid)
}

Write-Host 'Integration database users and Meilisearch index are ready.' -ForegroundColor Green
