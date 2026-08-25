. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

Wait-IntegrationHttp -Uri 'http://127.0.0.1:18080/'
Wait-IntegrationHttp -Uri 'http://127.0.0.1:18008/'

$blog = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18080/'
Assert-IntegrationStatus -Response $blog -Expected 200 -Name 'blog frontend'
if ($blog.Content -notmatch 'id="app"') { throw 'blog frontend HTML does not contain the Vue mount point' }

$admin = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18008/'
Assert-IntegrationStatus -Response $admin -Expected 200 -Name 'admin frontend'
if ($admin.Content -notmatch 'id="app"') { throw 'admin frontend HTML does not contain the Vue mount point' }

$cors = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18080/api/' -Method OPTIONS -Headers @{
    Origin = 'http://127.0.0.1:18080'
}
Assert-IntegrationStatus -Response $cors -Expected 204 -Name 'allowed CORS preflight'

$forbiddenCors = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18080/api/' -Method OPTIONS -Headers @{
    Origin = 'http://integration.invalid'
}
Assert-IntegrationStatus -Response $forbiddenCors -Expected 403 -Name 'forbidden CORS preflight'

$publicApi = @(
    @{ Name = 'blog home API'; Uri = 'http://127.0.0.1:18080/api/' },
    @{ Name = 'top and featured articles'; Uri = 'http://127.0.0.1:18080/api/articles/topAndFeatured' },
    @{ Name = 'article list'; Uri = 'http://127.0.0.1:18080/api/articles/all?current=1&size=10' },
    @{ Name = 'categories'; Uri = 'http://127.0.0.1:18080/api/categories/all' },
    @{ Name = 'tags'; Uri = 'http://127.0.0.1:18080/api/tags/all' },
    @{ Name = 'Meilisearch article search'; Uri = 'http://127.0.0.1:18080/api/articles/search?keywords=integration' }
)
foreach ($item in $publicApi) {
    $response = Invoke-IntegrationRequest -Uri $item.Uri
    [void](Assert-IntegrationApiSuccess -Response $response -Name $item.Name)
}

$loginBody = 'username=' + [uri]::EscapeDataString($env:E2E_ADMIN_EMAIL) + '&password=' + [uri]::EscapeDataString($env:E2E_ADMIN_PASSWORD)
$login = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18008/api/users/login' -Method POST -ContentType 'application/x-www-form-urlencoded' -Body $loginBody
$loginPayload = Assert-IntegrationApiSuccess -Response $login -Name 'admin login'
$token = [string]$loginPayload.data.token
if ([string]::IsNullOrWhiteSpace($token)) { throw 'admin login did not return a token' }
$authHeaders = @{ Authorization = "Bearer $token" }

$adminApi = @(
    @{ Name = 'admin home API'; Uri = 'http://127.0.0.1:18008/api/admin' },
    @{ Name = 'admin menu API'; Uri = 'http://127.0.0.1:18008/api/admin/user/menus' },
    @{ Name = 'website config API'; Uri = 'http://127.0.0.1:18008/api/admin/website/config' }
)
foreach ($item in $adminApi) {
    $response = Invoke-IntegrationRequest -Uri $item.Uri -Headers $authHeaders
    [void](Assert-IntegrationApiSuccess -Response $response -Name $item.Name)
}

$oldQQ = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18008/api/users/oauth/qq' -Method POST
Assert-IntegrationStatus -Response $oldQQ -Expected 404 -Name 'removed QQ OAuth endpoint'

$imageDirectory = Join-Path $script:IntegrationRepoRoot '.integration'
New-Item -ItemType Directory -Path $imageDirectory -Force | Out-Null
$imagePath = Join-Path $imageDirectory 'smoke.png'
$png = [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=')
[System.IO.File]::WriteAllBytes($imagePath, $png)
$uploadOutput = & curl.exe -sS -X POST -H "Authorization: Bearer $token" -F "file=@$imagePath;type=image/png" 'http://127.0.0.1:18008/api/admin/articles/images'
if ($LASTEXITCODE -ne 0) { throw 'MinIO upload request failed' }
$uploadPayload = $uploadOutput | ConvertFrom-Json
if (-not $uploadPayload.flag -or [string]::IsNullOrWhiteSpace([string]$uploadPayload.data)) {
    throw "MinIO upload failed: $uploadOutput"
}
$assetStatus = & curl.exe -sS -o NUL -w '%{http_code}' ([string]$uploadPayload.data)
if ($LASTEXITCODE -ne 0 -or [string]$assetStatus -ne '200') {
    throw "uploaded MinIO object is not publicly readable: HTTP $assetStatus"
}

$logout = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18008/api/users/logout' -Method POST -Headers $authHeaders
[void](Assert-IntegrationApiSuccess -Response $logout -Name 'logout')
$afterLogout = Invoke-IntegrationRequest -Uri 'http://127.0.0.1:18008/api/admin/user/menus' -Headers $authHeaders
Assert-IntegrationStatus -Response $afterLogout -Expected 401 -Name 'revoked admin session'

Write-Host 'Complete Caddy frontend/backend/DB/Redis/Meili/MinIO smoke test passed.' -ForegroundColor Green
