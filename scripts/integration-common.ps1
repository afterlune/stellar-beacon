$ErrorActionPreference = 'Stop'

$script:IntegrationRepoRoot = Split-Path -Parent $PSScriptRoot
$script:IntegrationProject = 'benetnasch-integration'
$script:IntegrationComposeFile = Join-Path $script:IntegrationRepoRoot 'docker-compose.integration.yaml'
$script:IntegrationEnvFile = Join-Path $script:IntegrationRepoRoot '.env.integration'
$script:IntegrationScopedEnvironmentNames = @(
    # Credentials and provider configuration must come only from .env.integration.
    'POSTGRES_PASSWORD', 'REDIS_PASSWORD', 'MEILI_MASTER_KEY', 'SMTP_PASSWORD',
    'MINIO_ROOT_USER', 'MINIO_ROOT_PASSWORD',
    'E2E_USER_EMAIL', 'E2E_USER_PASSWORD', 'E2E_ADMIN_EMAIL', 'E2E_ADMIN_PASSWORD',
    'E2E_ADMIN_TOKEN',
    'OPENAI_API_KEY', 'OPENAI_BASE_URL', 'OPENAI_MODEL',
    'ALIBAILIAN_API_KEY', 'ALIBAILIAN_BASE_URL', 'ALIBAILIAN_MODEL', 'ALIBAILIAN_MODEL2',
    'SGLANG_API_KEY', 'SGLANG_BASE_URL', 'SGLANG_MODEL', 'SGLANG_MODEL_ID',
    # Rollout flags and browser target variables must not leak from the parent shell.
    'BENETNASCH_AI_ENABLED', 'BENETNASCH_AI_OBSERVABILITY', 'BENETNASCH_AI_PROVIDER_PROBE',
    'BENETNASCH_AI_LOCAL_EMBEDDING', 'BENETNASCH_AI_LOCAL_EMBEDDING_MODEL_VERSION',
    'BENETNASCH_AI_LOCAL_EMBEDDING_INDEX_VERSION', 'BENETNASCH_AI_LOCAL_EMBEDDING_BATCH_SIZE',
    'BENETNASCH_AI_VISION', 'BENETNASCH_AI_ARTICLE_INDEXING',
    'BENETNASCH_AI_SPACE_COMPANION', 'BENETNASCH_AI_SPACE_COMPANION_PUBLISH',
    'BENETNASCH_SPACE_COMPANION_READ_TOKEN', 'BENETNASCH_SPACE_COMPANION_PUBLISH_TOKEN',
    'BENETNASCH_SPACE_COMPANION_AGENT_ID',
    'E2E_BASE_URL', 'E2E_REAL_INTEGRATION', 'E2E_ADMIN_ALLOW_LOGIN',
    'E2E_REQUIRE_AGENT_ROUTES', 'VITE_ADMIN_API_TARGET',
    'VITE_ADMIN_API_PRESERVE_API_PREFIX'
)
$script:IntegrationIgnoredEnvironmentNames = @(
    # These names can remain in a local file copied from the old deployment
    # template.  They are not part of the isolated Compose contract and must
    # be cleared without ever being forwarded to a child process.
    'ALIYUN_OSS_ACCESS_KEY_ID', 'ALIYUN_OSS_ACCESS_KEY_SECRET',
    'JWT_PRIVATE_KEY', 'JWT_PUBLIC_KEY', 'JWT_ALLOW_EPHEMERAL',
    'BENETNASCH_TRUSTED_PROXIES'
)
$script:IntegrationAllowedEnvironmentNames = @(
    $script:IntegrationScopedEnvironmentNames + 'INTEGRATION_COMPOSE_OVERRIDE'
)

function Import-IntegrationEnv {
    if (-not (Test-Path -LiteralPath $script:IntegrationEnvFile)) {
        throw "Missing $script:IntegrationEnvFile. Copy .env.integration.example to .env.integration first."
    }

    foreach ($name in ($script:IntegrationScopedEnvironmentNames + $script:IntegrationIgnoredEnvironmentNames)) {
        Remove-Item "Env:$name" -ErrorAction SilentlyContinue
    }

    # A local override may point at a stale prebuilt binary.  An explicit
    # source-compose request must win over that local convenience setting so
    # integration verification exercises the current repository contents.
    $useBaseCompose = $env:INTEGRATION_USE_BASE_COMPOSE -eq '1'
    if ($useBaseCompose) {
        Remove-Item Env:INTEGRATION_COMPOSE_OVERRIDE -ErrorAction SilentlyContinue
    }

    foreach ($line in Get-Content -LiteralPath $script:IntegrationEnvFile) {
        $trimmed = $line.Trim()
        if ($trimmed -eq '' -or $trimmed.StartsWith('#')) {
            continue
        }
        $parts = $trimmed -split '=', 2
        if ($parts.Count -ne 2) {
            continue
        }
        $name = $parts[0].Trim()
        if ($name -in $script:IntegrationIgnoredEnvironmentNames) {
            continue
        }
        if ($name -notin $script:IntegrationAllowedEnvironmentNames) {
            throw "Unsupported environment variable '$name' in .env.integration; use the documented integration allowlist."
        }
        if ($useBaseCompose -and $name -eq 'INTEGRATION_COMPOSE_OVERRIDE') {
            continue
        }
        $value = $parts[1].Trim()
        if (($value.StartsWith('"') -and $value.EndsWith('"')) -or
            ($value.StartsWith("'") -and $value.EndsWith("'"))) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        [Environment]::SetEnvironmentVariable($name, $value, 'Process')
    }
}

function Get-IntegrationComposeArgs {
    $composeArgs = @(
        'compose',
        '-p', $script:IntegrationProject,
        '--env-file', $script:IntegrationEnvFile,
        '-f', $script:IntegrationComposeFile
    )
    if (-not [string]::IsNullOrWhiteSpace($env:INTEGRATION_COMPOSE_OVERRIDE)) {
        $override = Join-Path $script:IntegrationRepoRoot $env:INTEGRATION_COMPOSE_OVERRIDE
        if (-not (Test-Path -LiteralPath $override)) {
            throw "Compose override does not exist: $override"
        }
        $composeArgs += @('-f', $override)
    }
    return $composeArgs
}

function Set-IntegrationLocalDatabaseTarget {
    # Host-run read-only commands must use the published integration port. The
    # explicit loopback override is guarded again inside the Go config package
    # and is never read from a user-provided remote target.
    $env:BENETNASCH_ENV = 'integration'
    $env:BENETNASCH_DATABASE_HOST = '127.0.0.1'
    $env:BENETNASCH_DATABASE_PORT = '15432'
}

function Invoke-IntegrationCompose {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $dockerArgs = @(Get-IntegrationComposeArgs) + $Arguments
    & docker @dockerArgs
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose failed with exit code $LASTEXITCODE"
    }
}

function Assert-IntegrationMigrationStatusCommand {
    $dockerArgs = @(Get-IntegrationComposeArgs) + @(
        'exec', '-T', 'backend',
        '/app/benetnasch', 'migrate', '--help'
    )
    $helpOutput = (& docker @dockerArgs 2>&1 | Out-String)
    $helpExitCode = $LASTEXITCODE
    if ($helpExitCode -ne 0 -or $helpOutput -notmatch '(?m)^\s+status\s+inspect migration state') {
        throw 'the running isolated backend image does not expose the read-only migrate status command; deploy the current image before inspecting or applying migrations.'
    }
}

function Wait-IntegrationHttp {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Uri,
        [int]$TimeoutSeconds = 120
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $Uri -TimeoutSec 5
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 400) {
                return
            }
        } catch {
            # The container is still starting.
        }
        Start-Sleep -Seconds 2
    } while ((Get-Date) -lt $deadline)
    throw "Timed out waiting for $Uri"
}

function Wait-IntegrationMeiliTask {
    param(
        [Parameter(Mandatory = $true)][string]$TaskUid,
        [int]$TimeoutSeconds = 60
    )

    $headers = @{ Authorization = "Bearer $env:MEILI_MASTER_KEY" }
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        Start-Sleep -Seconds 1
        $task = Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:17700/tasks/$TaskUid" -Headers $headers
        if ($task.status -eq 'succeeded') {
            return
        }
        if ($task.status -eq 'failed') {
            throw "Meilisearch task $TaskUid failed: $($task.error)"
        }
    } while ((Get-Date) -lt $deadline)
    throw "Timed out waiting for Meilisearch task $TaskUid"
}

function Assert-IntegrationStatus {
    param(
        [Parameter(Mandatory = $true)]$Response,
        [Parameter(Mandatory = $true)][int]$Expected,
        [Parameter(Mandatory = $true)][string]$Name
    )
    if ($Response.StatusCode -ne $Expected) {
        throw "$Name returned HTTP $($Response.StatusCode), expected $Expected. Body: $($Response.Content)"
    }
}

function Invoke-IntegrationRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [ValidateSet('GET', 'POST', 'PUT', 'DELETE', 'OPTIONS')][string]$Method = 'GET',
        [hashtable]$Headers,
        [string]$Body,
        [string]$ContentType = 'application/json'
    )

    try {
        $requestArgs = @{
            UseBasicParsing = $true
            Uri             = $Uri
            Method          = $Method
            TimeoutSec      = 30
        }
        if ($null -ne $Headers) { $requestArgs.Headers = $Headers }
        if ($null -ne $Body) {
            $requestArgs.Body = $Body
            $requestArgs.ContentType = $ContentType
        }
        $response = Invoke-WebRequest @requestArgs
        return [pscustomobject]@{
            StatusCode = [int]$response.StatusCode
            Content    = [string]$response.Content
        }
    } catch {
        $httpResponse = $_.Exception.Response
        if ($null -eq $httpResponse) {
            throw
        }
        if ($httpResponse -is [System.Net.Http.HttpResponseMessage]) {
            try {
                $content = $httpResponse.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            } catch {
                $content = ''
            }
        } else {
            try {
                $reader = New-Object System.IO.StreamReader($httpResponse.GetResponseStream())
                try {
                    $content = $reader.ReadToEnd()
                } finally {
                    $reader.Dispose()
                }
            } finally {
                # The response stream may already be closed by PowerShell.
            }
        }
        return [pscustomobject]@{
            StatusCode = [int]$httpResponse.StatusCode
            Content    = $content
        }
    }
}

function Convert-IntegrationJson {
    param([Parameter(Mandatory = $true)]$Response, [Parameter(Mandatory = $true)][string]$Name)
    try {
        return $Response.Content | ConvertFrom-Json
    } catch {
        throw "$Name did not return JSON: $($Response.Content)"
    }
}

function Assert-IntegrationApiSuccess {
    param([Parameter(Mandatory = $true)]$Response, [Parameter(Mandatory = $true)][string]$Name)
    Assert-IntegrationStatus -Response $Response -Expected 200 -Name $Name
    $payload = Convert-IntegrationJson -Response $Response -Name $Name
    if (-not $payload.flag) {
        throw "$Name returned an application failure: $($Response.Content)"
    }
    return $payload
}
