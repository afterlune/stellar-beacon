param(
    [ValidateSet('blog', 'admin', 'admin-next')]
    [string]$App
)

$ErrorActionPreference = 'Stop'

$budgets = @{
    'blog' = @{
        LargestJs = 900000
        TotalJs   = 1250000
        LargestCss = 500000
        TotalCss   = 600000
        TotalDist = 2000000
        Html      = 40000
    }
    'admin' = @{
        LargestJs = 2000000
        TotalJs   = 2900000
        LargestCss = 300000
        TotalCss   = 400000
        TotalDist = 3600000
        Html      = 40000
    }
    'admin-next' = @{
        LargestJs = 1150000
        TotalJs   = 1250000
        LargestCss = 500000
        TotalCss   = 550000
        TotalDist = 1750000
        Html      = 40000
    }
}

$apps = if ([string]::IsNullOrWhiteSpace($App)) { @('blog', 'admin', 'admin-next') } else { @($App) }

foreach ($name in $apps) {
    $dist = Join-Path $PSScriptRoot ("..\web\{0}\dist" -f $name)
    if (-not (Test-Path -LiteralPath $dist -PathType Container)) {
        throw "Missing frontend build output: $dist"
    }

    $jsFiles = @(Get-ChildItem -LiteralPath $dist -Recurse -File -Filter '*.js')
    $cssFiles = @(Get-ChildItem -LiteralPath $dist -Recurse -File -Filter '*.css')
    $htmlFile = Join-Path $dist 'index.html'
    if ($jsFiles.Count -eq 0 -or $cssFiles.Count -eq 0 -or -not (Test-Path -LiteralPath $htmlFile)) {
        throw "Incomplete frontend build output for ${name}: expected index.html, JavaScript and CSS assets."
    }

    $largestJs = [int64](($jsFiles | Measure-Object -Property Length -Maximum).Maximum)
    $totalJs = [int64](($jsFiles | Measure-Object -Property Length -Sum).Sum)
    $largestCss = [int64](($cssFiles | Measure-Object -Property Length -Maximum).Maximum)
    $totalCss = [int64](($cssFiles | Measure-Object -Property Length -Sum).Sum)
    $allFiles = @(Get-ChildItem -LiteralPath $dist -Recurse -File)
    $totalDist = [int64](($allFiles | Measure-Object -Property Length -Sum).Sum)
    $htmlSize = [int64](Get-Item -LiteralPath $htmlFile).Length
    $budget = $budgets[$name]

    $checks = @(
        @{ Name = 'largest JavaScript asset'; Actual = $largestJs; Limit = $budget.LargestJs },
        @{ Name = 'total JavaScript'; Actual = $totalJs; Limit = $budget.TotalJs },
        @{ Name = 'largest CSS asset'; Actual = $largestCss; Limit = $budget.LargestCss },
        @{ Name = 'total CSS'; Actual = $totalCss; Limit = $budget.TotalCss },
        @{ Name = 'total dist'; Actual = $totalDist; Limit = $budget.TotalDist },
        @{ Name = 'index.html'; Actual = $htmlSize; Limit = $budget.Html }
    )
    foreach ($check in $checks) {
        if ([int64]$check.Actual -gt [int64]$check.Limit) {
            throw "$name $($check.Name) is $($check.Actual) bytes, over the $($check.Limit)-byte budget."
        }
    }

    Write-Host ("{0}: JS {1}/{2} bytes, CSS {3}/{4} bytes, dist {5}/{6} bytes, HTML {7}/{8} bytes" -f `
        $name, $totalJs, $budget.TotalJs, $totalCss, $budget.TotalCss, $totalDist, $budget.TotalDist, $htmlSize, $budget.Html) -ForegroundColor Green
}
