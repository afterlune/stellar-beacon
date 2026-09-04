[CmdletBinding()]
param(
    [ValidateRange(1, 25)]
    [int]$PageSize = 1,
    [ValidateRange(0, [int]::MaxValue)]
    [int]$AfterArticleID = 0,
    [ValidateRange(0, [int]::MaxValue)]
    [int]$MaxArticles = 0,
    [ValidateRange(1, 3600)]
    [int]$TimeoutSeconds = 300
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv
Set-IntegrationLocalDatabaseTarget

# This wrapper intentionally exposes only the source-plan command. It is a
# host-side read-only query against the fixed loopback integration port; it
# never starts Compose, initializes a Provider, provisions Meilisearch, or
# passes any write-authorization flag to the Go command.
$arguments = @(
    'run', '.',
    'article-index', 'backfill', 'plan',
    '--page-size', $PageSize,
    '--after-article-id', $AfterArticleID,
    '--max-articles', $MaxArticles,
    '--timeout', ("{0}s" -f $TimeoutSeconds)
)
& go @arguments
if ($LASTEXITCODE -ne 0) {
    throw 'The read-only article-index plan command failed.'
}
