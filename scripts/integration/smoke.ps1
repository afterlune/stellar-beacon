. (Join-Path $PSScriptRoot 'common.ps1')
Import-IntegrationEnv

$blogBase = 'http://127.0.0.1:18080'
$adminBase = 'http://127.0.0.1:18008'

Wait-IntegrationHttp -Uri "$blogBase/"
Wait-IntegrationHttp -Uri "$adminBase/"

$ready = Invoke-IntegrationRequest -Uri "$blogBase/readyz"
Assert-IntegrationStatus -Response $ready -Expected 200 -Name 'backend readiness'
if ($ready.Content -notmatch '"ready":true') {
    throw "backend readiness did not report ready: $($ready.Content)"
}

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

$cors = Invoke-IntegrationRequest -Uri "$blogBase/api/v1/public/" -Method OPTIONS -Headers @{
    Origin = $blogBase
}
Assert-IntegrationStatus -Response $cors -Expected 204 -Name 'allowed CORS preflight'

$forbiddenCors = Invoke-IntegrationRequest -Uri "$blogBase/api/v1/public/" -Method OPTIONS -Headers @{
    Origin = 'http://integration.invalid'
}
Assert-IntegrationStatus -Response $forbiddenCors -Expected 403 -Name 'forbidden CORS preflight'

$publicApi = @(
    @{ Name = 'blog home API'; Uri = "$blogBase/api/v1/public/" },
    @{ Name = 'top and featured articles'; Uri = "$blogBase/api/v1/public/articles/featured" },
    @{ Name = 'article list'; Uri = "$blogBase/api/v1/public/articles?current=1&size=10" },
    @{ Name = 'categories'; Uri = "$blogBase/api/v1/public/categories" },
    @{ Name = 'tags'; Uri = "$blogBase/api/v1/public/tags" },
    @{ Name = 'Meilisearch article search'; Uri = "$blogBase/api/v1/public/articles/search?keywords=integration" }
)
foreach ($item in $publicApi) {
    $response = Invoke-IntegrationRequest -Uri $item.Uri
    [void](Assert-IntegrationApiSuccess -Response $response -Name $item.Name)
}

$publicArticleListResponse = Invoke-IntegrationRequest -Uri "$blogBase/api/v1/public/articles?current=1&size=10"
$publicArticleList = Assert-IntegrationApiSuccess -Response $publicArticleListResponse -Name 'public article list for content analytics'
$publicArticle = @($publicArticleList.data.items) | Where-Object { [int]$_.status -eq 1 } | Select-Object -First 1
if ($null -eq $publicArticle) {
    throw 'content analytics smoke requires at least one published article'
}
$contentArticleId = [int]$publicArticle.id
$articleResponse = Invoke-IntegrationRequest -Uri "$blogBase/api/v1/public/articles/$contentArticleId"
$articleDetail = Assert-IntegrationApiSuccess -Response $articleResponse -Name 'public article detail for content analytics'

$readSessionId = [guid]::NewGuid().ToString()
$readSessionBody = @{
    sessionId = $readSessionId
    activeMs = 5000
    maxScrollPercent = 95
} | ConvertTo-Json -Compress
for ($attempt = 1; $attempt -le 2; $attempt++) {
    $readSession = Invoke-IntegrationRequest -Uri "$blogBase/api/v1/public/articles/$contentArticleId/read-sessions" -Method POST -ContentType 'application/json' -Body $readSessionBody
    [void](Assert-IntegrationApiSuccess -Response $readSession -Name "content analytics read session attempt $attempt")
}

$continuationEvents = @(
    @{ eventType = 'series_impression' },
    @{ eventType = 'related_impression' }
)
$seriesId = [int]$articleDetail.data.seriesId
if ($seriesId -gt 0) {
    $continuationEvents += @{ eventType = 'series_click'; targetType = 'series'; targetId = $seriesId; placement = 'series_index'; position = 0 }
}
$relatedTarget = @($articleDetail.data.relatedArticles) | Where-Object { $null -ne $_ -and [int]$_.id -gt 0 } | Select-Object -First 1
if ($null -ne $relatedTarget) {
    $continuationEvents += @{ eventType = 'related_click'; targetType = 'article'; targetId = [int]$relatedTarget.id; placement = 'related'; position = 1 }
}
foreach ($event in $continuationEvents) {
    $continuationBody = $event | ConvertTo-Json -Compress
    $continuation = Invoke-IntegrationRequest -Uri "$blogBase/api/v1/public/articles/$contentArticleId/continuation-events" -Method POST -ContentType 'application/json' -Body $continuationBody
    [void](Assert-IntegrationApiSuccess -Response $continuation -Name "content analytics continuation event $($event.eventType)")
}
$loginBody = 'username=' + [uri]::EscapeDataString($env:E2E_ADMIN_EMAIL) + '&password=' + [uri]::EscapeDataString($env:E2E_ADMIN_PASSWORD)
$login = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/auth/login" -Method POST -ContentType 'application/x-www-form-urlencoded' -Body $loginBody
$loginPayload = Assert-IntegrationApiSuccess -Response $login -Name 'admin login'
$token = [string]$loginPayload.data.token
if ([string]::IsNullOrWhiteSpace($token)) { throw 'admin login did not return a token' }
$authHeaders = @{ Authorization = "Bearer $token" }

$adminApi = @(
    @{ Name = 'admin home API'; Uri = "$adminBase/api/v1/admin/dashboard" },
    @{ Name = 'admin menu API'; Uri = "$adminBase/api/v1/admin/me/menu" },
    @{ Name = 'website config API'; Uri = "$adminBase/api/v1/admin/site" },
    @{ Name = 'job targets API'; Uri = "$adminBase/api/v1/admin/jobs/targets" },
    @{ Name = 'content analytics overview API'; Uri = "$adminBase/api/v1/admin/content/analytics?range=7d" },
    @{ Name = 'content analytics article list API'; Uri = "$adminBase/api/v1/admin/content/analytics/articles?range=7d&sort=views&current=1&size=10" },
    @{ Name = 'content analytics continuation targets API'; Uri = "$adminBase/api/v1/admin/content/analytics/continuation-targets?range=7d&sort=clicks&current=1&size=10" }
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

$contentAnalyticsDetailResponse = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/admin/content/analytics/articles/${contentArticleId}?range=7d" -Headers $authHeaders
$contentAnalyticsDetail = Assert-IntegrationApiSuccess -Response $contentAnalyticsDetailResponse -Name 'content analytics article detail API'
if ([int]$contentAnalyticsDetail.data.overview.views -lt 1 -or [int]$contentAnalyticsDetail.data.overview.effectiveSessions -lt 1) {
    throw "content analytics did not persist the smoke reading session: $($contentAnalyticsDetailResponse.Content)"
}
$continuationOverview = $contentAnalyticsDetail.data.overview.continuation
if ([int]$continuationOverview.seriesImpressions -lt 1 -or [int]$continuationOverview.relatedImpressions -lt 1) {
    throw "content analytics did not persist the smoke continuation impressions: $($contentAnalyticsDetailResponse.Content)"
}
if ($seriesId -gt 0 -and [int]$continuationOverview.seriesClicks -lt 1) {
    throw "content analytics did not persist the smoke series continuation click: $($contentAnalyticsDetailResponse.Content)"
}
if ($null -ne $relatedTarget -and [int]$continuationOverview.relatedClicks -lt 1) {
    throw "content analytics did not persist the smoke related continuation click: $($contentAnalyticsDetailResponse.Content)"
}if ($seriesId -gt 0 -or $null -ne $relatedTarget) {
    $continuationTargetResponse = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/admin/content/analytics/continuation-targets?range=7d&sort=clicks&current=1&size=10&sourceArticleId=$contentArticleId" -Headers $authHeaders
    $continuationTargetPayload = Assert-IntegrationApiSuccess -Response $continuationTargetResponse -Name 'content analytics continuation target detail API'
    if ([int]$continuationTargetPayload.data.total -lt 1) {
        throw "content analytics did not persist attributed continuation targets: $($continuationTargetResponse.Content)"
    }
}

$adminListApi = @(
    @{ Name = 'admin article list'; Uri = "$adminBase/api/v1/admin/articles?current=1&size=10" },
    @{ Name = 'admin category list'; Uri = "$adminBase/api/v1/admin/categories?current=1&size=10" },
    @{ Name = 'admin tag list'; Uri = "$adminBase/api/v1/admin/tags?current=1&size=10" },
    @{ Name = 'admin user list'; Uri = "$adminBase/api/v1/admin/users?current=1&size=10" }
)
foreach ($item in $adminListApi) {
    $response = Invoke-IntegrationRequest -Uri $item.Uri -Headers $authHeaders
    $payload = Assert-IntegrationApiSuccess -Response $response -Name $item.Name
    if ($null -eq $payload.data -or @($payload.data.items).Count -eq 0) {
        throw "$($item.Name) returned no seeded records. Body: $($response.Content)"
    }
}

$operationLogsBeforeResponse = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/admin/logs/operations?current=1&size=10" -Headers $authHeaders
$operationLogsBefore = Assert-IntegrationApiSuccess -Response $operationLogsBeforeResponse -Name 'operation log baseline'
$operationLogCountBefore = [int]$operationLogsBefore.data.total

$oldQQ = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/auth/oauth/qq" -Method POST
Assert-IntegrationStatus -Response $oldQQ -Expected 404 -Name 'removed QQ OAuth endpoint'

$imageDirectory = Join-Path $script:IntegrationRepoRoot '.integration'
New-Item -ItemType Directory -Path $imageDirectory -Force | Out-Null
$imagePath = Join-Path $imageDirectory 'smoke.png'
$png = [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=')
[System.IO.File]::WriteAllBytes($imagePath, $png)
$uploadOutput = & curl.exe -sS -X POST -H "Authorization: Bearer $token" -F "file=@$imagePath;type=image/png" "$adminBase/api/v1/admin/articles/images"
if ($LASTEXITCODE -ne 0) { throw 'MinIO upload request failed' }
$uploadPayload = $uploadOutput | ConvertFrom-Json
if (($uploadPayload.code -ne 'OK' -and -not $uploadPayload.flag) -or [string]::IsNullOrWhiteSpace([string]$uploadPayload.data)) {
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
    $operationLogsAfterResponse = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/admin/logs/operations?current=1&size=10" -Headers $authHeaders
    if ($operationLogsAfterResponse.StatusCode -eq 200) {
        $operationLogsAfter = Convert-IntegrationJson -Response $operationLogsAfterResponse -Name 'operation log after upload'
        if (($operationLogsAfter.code -eq 'OK' -or $operationLogsAfter.flag) -and [int]$operationLogsAfter.data.total -gt $operationLogCountBefore) {
            break
        }
    }
} while ((Get-Date) -lt $operationLogDeadline)

if ($null -eq $operationLogsAfter -or ($operationLogsAfter.code -ne 'OK' -and -not $operationLogsAfter.flag) -or [int]$operationLogsAfter.data.total -le $operationLogCountBefore) {
    throw "operation log was not persisted after upload. Before=$operationLogCountBefore"
}
$uploadLog = @($operationLogsAfter.data.items) | Where-Object { $_.optUri -match '/articles/images' } | Select-Object -First 1
if ($null -eq $uploadLog -or [string]$uploadLog.requestParam -notmatch 'request body omitted') {
    throw 'upload operation log did not contain the sanitized request marker'
}

$logout = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/auth/logout" -Method POST -Headers $authHeaders
[void](Assert-IntegrationApiSuccess -Response $logout -Name 'logout')
$afterLogout = Invoke-IntegrationRequest -Uri "$adminBase/api/v1/admin/me/menu" -Headers $authHeaders
Assert-IntegrationStatus -Response $afterLogout -Expected 401 -Name 'revoked admin session'

Write-Host 'Complete Caddy frontend/backend/DB/Redis/Meili/MinIO smoke test passed.' -ForegroundColor Green
