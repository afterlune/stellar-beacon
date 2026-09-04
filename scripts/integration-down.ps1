param(
    [switch]$AllowContainerChanges,
    [switch]$RemoveVolumes
)

$ErrorActionPreference = 'Stop'

if (-not $AllowContainerChanges) {
    throw 'Refusing to stop or remove the isolated Compose project. Re-run with -AllowContainerChanges during an approved integration window.'
}

. (Join-Path $PSScriptRoot 'integration-common.ps1')
Import-IntegrationEnv

$downArgs = @('down', '--remove-orphans')
if ($RemoveVolumes) {
    $downArgs += '-v'
}
Invoke-IntegrationCompose -Arguments $downArgs
Write-Host 'Only the isolated integration Compose project was stopped.' -ForegroundColor Green
