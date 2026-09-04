[CmdletBinding()]
param(
    [switch]$SkipGo,
    [switch]$SkipFrontend,
    [switch]$BuildFrontend,
    [switch]$RunBrowserBaseline
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($SkipFrontend -and ($BuildFrontend -or $RunBrowserBaseline)) {
    throw 'SkipFrontend cannot be combined with BuildFrontend or RunBrowserBaseline.'
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$bashChecks = @(
    'scripts/check-toolchain-version.sh',
    'scripts/check-compose-image-pins.sh',
    'scripts/check-compose-provider-env.sh',
    'scripts/check-env-template-safety.sh',
    'scripts/check-qwen-overlay-safety.sh',
    'scripts/check-integration-safety-gates.sh',
    'scripts/check-no-dynamic-sql.sh',
    'scripts/check-service-db-boundary.sh',
    'scripts/check-service-infra-boundary.sh',
    'scripts/check-repository-session-boundary.sh',
    'scripts/check-service-constructor-boundary.sh',
    'scripts/check-application-ai-boundary.sh',
    'scripts/check-application-boundary.sh',
    'scripts/check-article-index-write-gates.sh',
    'scripts/check-log-sanitization.sh',
    'scripts/check-migration-write-gate.sh',
    'scripts/check-production-backup-safety.sh',
    'scripts/check-production-release-safety.sh',
    'scripts/check-no-tencent-captcha.sh',
    'scripts/check-admin-menu-components.sh',
    'scripts/check-no-openpgp.sh'
)

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )

    Write-Host ("`n> {0} {1}" -f $FilePath, ($Arguments -join ' ')) -ForegroundColor DarkCyan
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code $LASTEXITCODE`: $FilePath $($Arguments -join ' ')"
    }
}

function Invoke-NpmApp {
    param(
        [Parameter(Mandatory = $true)][string]$App,
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )

    $appRoot = Join-Path $repoRoot (Join-Path 'web' $App)
    if (-not (Test-Path -LiteralPath $appRoot -PathType Container)) {
        throw "Frontend directory does not exist: $appRoot"
    }
    Push-Location $appRoot
    try {
        Invoke-Checked -FilePath 'npm' -Arguments $Arguments
    } finally {
        Pop-Location
    }
}

Push-Location $repoRoot
try {
    Invoke-Checked -FilePath 'git' -Arguments @('diff', '--check')

    $legacyDirectories = @(
        (Join-Path $repoRoot 'app/infra/SearchEngines'),
        (Join-Path $repoRoot 'app/infra/zlog')
    )
    foreach ($directory in $legacyDirectories) {
        if (Test-Path -LiteralPath $directory) {
            throw "Legacy infrastructure directory remains: $directory"
        }
    }

    $rootErrorLogs = @(Get-ChildItem -LiteralPath $repoRoot -File -Filter 'error_*.log' -ErrorAction SilentlyContinue)
    $appRoot = Join-Path $repoRoot 'app'
    $appErrorLogs = @()
    if (Test-Path -LiteralPath $appRoot -PathType Container) {
        $appErrorLogs = @(Get-ChildItem -LiteralPath $appRoot -Recurse -File -Filter 'error_*.log' -ErrorAction SilentlyContinue)
    }
    $unexpectedLogs = @($rootErrorLogs + $appErrorLogs)
    if ($unexpectedLogs.Count -gt 0) {
        throw "Legacy/generated error logs remain outside the managed resource log directory: $($unexpectedLogs.FullName -join ', ')"
    }

    if (-not $SkipGo) {
        Invoke-Checked -FilePath 'go' -Arguments @('test', '-count=1', './...')
        Invoke-Checked -FilePath 'go' -Arguments @('vet', './...')
        foreach ($script in $bashChecks) {
            Invoke-Checked -FilePath 'bash' -Arguments @($script)
        }
    }

    if (-not $SkipFrontend) {
        if ($BuildFrontend) {
            foreach ($app in @('blog', 'admin', 'admin-next')) {
                Invoke-NpmApp -App $app -Arguments @('run', 'build')
            }
        }
        Invoke-Checked -FilePath 'pwsh' -Arguments @('-NoProfile', '-File', 'scripts/check-frontend-budgets.ps1')
        Invoke-Checked -FilePath 'pwsh' -Arguments @('-NoProfile', '-File', 'scripts/check-admin-next-release-assets.ps1')
        if ($RunBrowserBaseline) {
            Invoke-NpmApp -App 'blog' -Arguments @('run', 'test:e2e:baseline', '--', '--workers=1')
            Invoke-NpmApp -App 'admin-next' -Arguments @('run', 'test:e2e:baseline', '--', '--workers=1')
        }
    }

    Write-Host "`nSafe preflight passed." -ForegroundColor Green
    Write-Host 'This command does not start/stop containers, execute database migrations, provision or write Meilisearch, or switch a Caddy static directory.' -ForegroundColor Yellow
    Write-Host 'Real isolated browser/RBAC, index cutover, backup/restore, and production release gates remain separate operations.' -ForegroundColor Yellow
} finally {
    Pop-Location
}
