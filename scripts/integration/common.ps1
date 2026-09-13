$ErrorActionPreference = 'Stop'

$script:IntegrationRepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$script:IntegrationProject = if ([string]::IsNullOrWhiteSpace($env:INTEGRATION_COMPOSE_PROJECT)) {
    'stellar-beacon-integration-v17'
} else {
    $env:INTEGRATION_COMPOSE_PROJECT
}
$script:IntegrationComposeFile = Join-Path $script:IntegrationRepoRoot 'deploy/compose/integration.yaml'
$script:IntegrationEnvFile = Join-Path $script:IntegrationRepoRoot '.env.integration'

function Import-IntegrationEnv {
    if (-not (Test-Path -LiteralPath $script:IntegrationEnvFile)) {
        throw "Missing $script:IntegrationEnvFile. Copy .env.integration.example to .env.integration first."
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
        '-f', $script:IntegrationComposeFile,
        '--project-directory', $script:IntegrationRepoRoot
    )
    if (-not [string]::IsNullOrWhiteSpace($env:INTEGRATION_COMPOSE_OVERRIDE)) {
        $override = Join-Path $script:IntegrationRepoRoot $env:INTEGRATION_COMPOSE_OVERRIDE
        if (-not (Test-Path -LiteralPath $override)) {
            Write-Warning "Skipping missing optional compose override: $override"
            return $composeArgs
        }
        $composeArgs += @('-f', $override)
    }
    return $composeArgs
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
    if ($payload.code -ne 'OK' -and -not $payload.flag) {
        throw "$Name returned an application failure: $($Response.Content)"
    }
    return $payload
}
