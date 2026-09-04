param(
    [string]$BaseUrl = 'http://127.0.0.1:18018',
    [ValidateSet('chat', 'vision', 'embedding')]
    [string]$UseCase = 'vision',
    [switch]$UseLocalSGLang,
    [switch]$AllowExternalProviderCalls,
    [switch]$AllowWrites
)

$ErrorActionPreference = 'Stop'

if (-not $AllowExternalProviderCalls -or -not $AllowWrites) {
    throw 'Refusing to call an external model or append an operation log. Re-run with -AllowExternalProviderCalls -AllowWrites during an approved isolated smoke window.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')

# Import-IntegrationEnv clears credentials, rollout flags and browser targets
# before loading the explicitly selected isolated env file.
Import-IntegrationEnv

$uri = [Uri]$BaseUrl
if ($uri.Scheme -ne 'http' -or $uri.Host -notin @('127.0.0.1', 'localhost') -or $uri.Port -ne 18018) {
    throw 'BaseUrl must point to isolated admin-next loopback port 18018; refusing to call another target.'
}
$UseCase = $UseCase.ToLowerInvariant()
if ($UseCase -eq 'embedding') {
	if ($UseLocalSGLang) {
		& (Join-Path $PSScriptRoot 'check-qwen-memory.ps1') -RequireRunningContainer
		$composeArgs = @(Get-IntegrationComposeArgs)
		$backendEnvCheck = $composeArgs + @(
			'exec', '-T', 'backend', '/bin/sh', '-c',
			'test "$BENETNASCH_AI_LOCAL_EMBEDDING" = "true" && test "$BENETNASCH_AI_ENABLED" = "true" && test "$BENETNASCH_AI_PROVIDER_PROBE" = "true" && test "$SGLANG_BASE_URL" = "http://qwen-embedding:30000/v1" && test "$SGLANG_MODEL" = "Qwen/Qwen3-Embedding-0.6B"'
		)
		& docker @backendEnvCheck 2>$null
		if ($LASTEXITCODE -ne 0) {
            throw 'running isolated backend is not using the explicit local Qwen/SGLang overlay; start it with docker-compose.integration.qwen.yaml and --profile qwen-low-memory first.'
		}
	} elseif ([string]::IsNullOrWhiteSpace($env:ALIBAILIAN_API_KEY)) {
		throw 'ALIBAILIAN_API_KEY must be set in .env.integration for the configured embedding route; the value is never printed.'
	}
} elseif ($UseLocalSGLang) {
	throw '-UseLocalSGLang is only valid with -UseCase embedding.'
} elseif ([string]::IsNullOrWhiteSpace($env:OPENAI_API_KEY)) {
    throw 'OPENAI_API_KEY must be set in .env.integration; the value is never printed.'
}

function Assert-EnabledFlag {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $parsed = $false
    if (-not [bool]::TryParse([string](Get-Item "Env:$Name" -ErrorAction SilentlyContinue).Value, [ref]$parsed) -or -not $parsed) {
        throw "$Name must be explicitly set to true in .env.integration before provider smoke; the script never changes rollout flags."
    }
}

if (-not $UseLocalSGLang) {
	Assert-EnabledFlag -Name 'BENETNASCH_AI_ENABLED'
	Assert-EnabledFlag -Name 'BENETNASCH_AI_PROVIDER_PROBE'
}

function Get-AdminToken {
    if ([string]::IsNullOrWhiteSpace($env:E2E_ADMIN_EMAIL) -or [string]::IsNullOrWhiteSpace($env:E2E_ADMIN_PASSWORD)) {
        throw 'isolated admin credentials must be set in .env.integration; credentials are never printed.'
    }

    $body = @{ username = $env:E2E_ADMIN_EMAIL; password = $env:E2E_ADMIN_PASSWORD } | ConvertTo-Json -Compress
    $response = Invoke-IntegrationRequest -Uri "$($uri.AbsoluteUri.TrimEnd('/'))/api/users/login" -Method POST -Body $body
    if ($response.StatusCode -ne 200) {
        throw 'isolated admin login failed while preparing provider smoke.'
    }
    try {
        $payload = $response.Content | ConvertFrom-Json
    } catch {
        throw 'isolated admin login returned malformed JSON while preparing provider smoke.'
    }
    if ($null -eq $payload -or $payload.flag -ne $true -or [int]$payload.code -ne 20000) {
        throw 'isolated admin login returned an unsuccessful result while preparing provider smoke.'
    }
    if ($payload.data -is [string] -and -not [string]::IsNullOrWhiteSpace([string]$payload.data)) {
        return [string]$payload.data
    }
    foreach ($field in @('token', 'accessToken', 'access_token')) {
        $property = $payload.data.PSObject.Properties[$field]
        if ($null -ne $property -and -not [string]::IsNullOrWhiteSpace([string]$property.Value)) {
            return [string]$property.Value
        }
    }
    throw 'isolated admin login did not return a token for provider smoke.'
}

$token = Get-AdminToken
$body = @{ useCase = $UseCase } | ConvertTo-Json -Compress
$headers = @{ Authorization = "Bearer $token" }
$response = Invoke-IntegrationRequest `
    -Uri "$($uri.AbsoluteUri.TrimEnd('/'))/api/admin/ai/providers/test" `
    -Method POST `
    -Headers $headers `
    -Body $body

if ($response.StatusCode -ne 200) {
    throw 'provider smoke returned a non-success HTTP status.'
}
try {
    $payload = $response.Content | ConvertFrom-Json
} catch {
    throw 'provider smoke returned malformed JSON.'
}
if ($null -eq $payload -or $payload.flag -ne $true -or [int]$payload.code -ne 20000) {
    throw 'provider smoke did not pass; inspect the isolated backend logs without exposing the response body.'
}
$result = $payload.data
if ($null -eq $result -or [string]$result.useCase -ne $UseCase -or $result.reachable -ne $true) {
    throw 'provider smoke returned an incomplete capability result.'
}
if ($UseCase -eq 'embedding') {
    if ($result.embedding -ne $true -or $result.generate -eq $true -or $result.streaming -eq $true) {
        throw 'embedding provider smoke did not report exactly the embedding capability.'
    }
} elseif ($result.generate -ne $true -or $result.streaming -ne $true -or $result.embedding -eq $true) {
    throw 'chat/vision provider smoke did not report the expected generation and streaming capabilities.'
}

foreach ($field in @('apiKey', 'password', 'secret', 'authorization', 'prompt', 'request', 'response')) {
    if ($null -ne $result.PSObject.Properties[$field]) {
        throw 'provider smoke response contains a forbidden sensitive field.'
    }
}

Write-Host "isolated $UseCase Provider smoke passed; response fields were checked for credential and payload leakage." -ForegroundColor Green
