. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$blogBase = 'http://127.0.0.1:18080'
$adminBase = 'http://127.0.0.1:18008'

Wait-IntegrationHttp -Uri "$blogBase/"
Wait-IntegrationHttp -Uri "$adminBase/"

$blog = Invoke-IntegrationRequest -Uri "$blogBase/"
Assert-IntegrationStatus -Response $blog -Expected 200 -Name 'blog frontend'
if ($blog.Content -notmatch 'id="app"') { throw 'blog frontend HTML does not contain the Vue mount point' }
if ($blog.Content -match 'TencentCaptcha|TCaptcha\.js|captcha\.qq\.com|TENCENT_CAPTCHA') {
    throw 'blog frontend still references Tencent CAPTCHA'
}

$admin = Invoke-IntegrationRequest -Uri "$adminBase/"
Assert-IntegrationStatus -Response $admin -Expected 200 -Name 'admin frontend'
if ($admin.Content -notmatch 'id="app"') { throw 'admin frontend HTML does not contain the Vue mount point' }
if ($admin.Content -match 'TencentCaptcha|TCaptcha\.js|captcha\.qq\.com|TENCENT_CAPTCHA') {
    throw 'admin frontend still references Tencent CAPTCHA'
}

$cors = Invoke-IntegrationRequest -Uri "$blogBase/api/" -Method OPTIONS -Headers @{
    Origin = $blogBase
}
Assert-IntegrationStatus -Response $cors -Expected 204 -Name 'allowed CORS preflight'

$forbiddenCors = Invoke-IntegrationRequest -Uri "$blogBase/api/" -Method OPTIONS -Headers @{
    Origin = 'http://integration.invalid'
}
Assert-IntegrationStatus -Response $forbiddenCors -Expected 403 -Name 'forbidden CORS preflight'

$publicApi = @(
    @{ Name = 'blog home API'; Uri = "$blogBase/api/" },
    @{ Name = 'top and featured articles'; Uri = "$blogBase/api/articles/topAndFeatured" },
    @{ Name = 'article list'; Uri = "$blogBase/api/articles/all?current=1&size=10" },
    @{ Name = 'categories'; Uri = "$blogBase/api/categories/all" },
    @{ Name = 'tags'; Uri = "$blogBase/api/tags/all" },
    @{ Name = 'Meilisearch article search'; Uri = "$blogBase/api/articles/search?keywords=integration" }
)
foreach ($item in $publicApi) {
    $response = Invoke-IntegrationRequest -Uri $item.Uri
    [void](Assert-IntegrationApiSuccess -Response $response -Name $item.Name)
}

$loginBody = 'username=' + [uri]::EscapeDataString($env:E2E_ADMIN_EMAIL) + '&password=' + [uri]::EscapeDataString($env:E2E_ADMIN_PASSWORD)
$login = Invoke-IntegrationRequest -Uri "$adminBase/api/users/login" -Method POST -ContentType 'application/x-www-form-urlencoded' -Body $loginBody
$loginPayload = Assert-IntegrationApiSuccess -Response $login -Name 'admin login'
$token = [string]$loginPayload.data.token
if ([string]::IsNullOrWhiteSpace($token)) { throw 'admin login did not return a token' }
$authHeaders = @{ Authorization = "Bearer $token" }

$adminApi = @(
    @{ Name = 'admin home API'; Uri = "$adminBase/api/admin" },
    @{ Name = 'admin menu API'; Uri = "$adminBase/api/admin/user/menus" },
    @{ Name = 'website config API'; Uri = "$adminBase/api/admin/website/config" }
)
foreach ($item in $adminApi) {
    $response = Invoke-IntegrationRequest -Uri $item.Uri -Headers $authHeaders
    $payload = Assert-IntegrationApiSuccess -Response $response -Name $item.Name
    if ($item.Name -eq 'admin menu API') {
        $menus = @($payload.data)
        if ($menus.Count -eq 0) { throw 'admin menu API returned no menus' }
        foreach ($menu in $menus) {
            if ([string]::IsNullOrWhiteSpace([string]$menu.path) -or @($menu.children).Count -eq 0) {
                throw "admin menu API returned an invalid top-level route: $($response.Content)"
            }
            foreach ($child in @($menu.children)) {
                if ($null -eq $child.path -or [string]::IsNullOrWhiteSpace([string]$child.component)) {
                    throw "admin menu API returned an invalid child route: $($response.Content)"
                }
            }
        }
    }
}

$adminListApi = @(
    @{ Name = 'admin article list'; Uri = "$adminBase/api/admin/articles?current=1&size=10" },
    @{ Name = 'admin category list'; Uri = "$adminBase/api/admin/categories?current=1&size=10" },
    @{ Name = 'admin tag list'; Uri = "$adminBase/api/admin/tags?current=1&size=10" },
    @{ Name = 'admin user list'; Uri = "$adminBase/api/admin/users?current=1&size=10" }
)
foreach ($item in $adminListApi) {
    $response = Invoke-IntegrationRequest -Uri $item.Uri -Headers $authHeaders
    $payload = Assert-IntegrationApiSuccess -Response $response -Name $item.Name
    if ($null -eq $payload.data -or @($payload.data.records).Count -eq 0) {
        throw "$($item.Name) returned no seeded records. Body: $($response.Content)"
    }
}

$operationLogsBeforeResponse = Invoke-IntegrationRequest -Uri "$adminBase/api/admin/operation/logs?current=1&size=10" -Headers $authHeaders
$operationLogsBefore = Assert-IntegrationApiSuccess -Response $operationLogsBeforeResponse -Name 'operation log baseline'
$operationLogCountBefore = [int]$operationLogsBefore.data.count

$oldQQ = Invoke-IntegrationRequest -Uri "$adminBase/api/users/oauth/qq" -Method POST
Assert-IntegrationStatus -Response $oldQQ -Expected 404 -Name 'removed QQ OAuth endpoint'

$imageDirectory = Join-Path $script:IntegrationRepoRoot '.integration'
New-Item -ItemType Directory -Path $imageDirectory -Force | Out-Null
$imagePath = Join-Path $imageDirectory 'smoke.png'
$png = [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=')
[System.IO.File]::WriteAllBytes($imagePath, $png)
$uploadOutput = & curl.exe -sS -X POST -H "Authorization: Bearer $token" -F "file=@$imagePath;type=image/png" "$adminBase/api/admin/articles/images"
if ($LASTEXITCODE -ne 0) { throw 'MinIO upload request failed' }
$uploadPayload = $uploadOutput | ConvertFrom-Json
if (-not $uploadPayload.flag -or [string]::IsNullOrWhiteSpace([string]$uploadPayload.data)) {
    throw "MinIO upload failed: $uploadOutput"
}
$assetStatus = & curl.exe -sS -o NUL -w '%{http_code}' ([string]$uploadPayload.data)
if ($LASTEXITCODE -ne 0 -or [string]$assetStatus -ne '200') {
    throw "uploaded MinIO object is not publicly readable: HTTP $assetStatus"
}

$operationLogsAfter = $null
$operationLogDeadline = (Get-Date).AddSeconds(10)
do {
    Start-Sleep -Milliseconds 500
    $operationLogsAfterResponse = Invoke-IntegrationRequest -Uri "$adminBase/api/admin/operation/logs?current=1&size=10" -Headers $authHeaders
    if ($operationLogsAfterResponse.StatusCode -eq 200) {
        $operationLogsAfter = Convert-IntegrationJson -Response $operationLogsAfterResponse -Name 'operation log after upload'
        if ($operationLogsAfter.flag -and [int]$operationLogsAfter.data.count -gt $operationLogCountBefore) {
            break
        }
    }
} while ((Get-Date) -lt $operationLogDeadline)

if ($null -eq $operationLogsAfter -or -not $operationLogsAfter.flag -or [int]$operationLogsAfter.data.count -le $operationLogCountBefore) {
    throw "operation log was not persisted after upload. Before=$operationLogCountBefore"
}
$uploadLog = @($operationLogsAfter.data.records) | Where-Object { $_.optUri -match '/articles/images' } | Select-Object -First 1
if ($null -eq $uploadLog -or [string]$uploadLog.requestParam -notmatch 'request body omitted') {
    throw 'upload operation log did not contain the sanitized request marker'
}

$logout = Invoke-IntegrationRequest -Uri "$adminBase/api/users/logout" -Method POST -Headers $authHeaders
[void](Assert-IntegrationApiSuccess -Response $logout -Name 'logout')
$afterLogout = Invoke-IntegrationRequest -Uri "$adminBase/api/admin/user/menus" -Headers $authHeaders
Assert-IntegrationStatus -Response $afterLogout -Expected 401 -Name 'revoked admin session'

Write-Host 'Complete Caddy frontend/backend/DB/Redis/Meili/MinIO smoke test passed.' -ForegroundColor Green
