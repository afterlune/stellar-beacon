param(
    [switch]$RemoveVolumes
)

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$downArgs = @('down', '--remove-orphans')
if ($RemoveVolumes) {
    $downArgs += '-v'
}
Invoke-IntegrationCompose -Arguments $downArgs
Write-Host 'Only the isolated integration Compose project was stopped.' -ForegroundColor Green
