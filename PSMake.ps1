Param (
    [string]$version = "dev"
)
$ErrorActionPreference = "Stop"

# Required tools:
# https://github.com/google/go-licenses - for license reporting
# https://github.com/billgraziano/blackfriday - for converting documenation markdown to HTML

Write-Output "Running PSBuild.ps1..."
$target=".\deploy\isitsql"
Write-Output "Target: $target"

# $now = Get-Date -UFormat "%Y-%m-%d_%T"
$sha1 = (git describe --tags --dirty --always).Trim()
$gitBranch = (git rev-parse --abbrev-ref HEAD)
if ($gitBranch -ne "master") {
    $sha1 = $sha1 + "-$($gitBranch)"
}
$semver = ConvertTo-Semver -Version $version
Write-Host "" 
Write-Host "Version:    $version ($($semver.ToString()))"  -ForegroundColor Green
Write-Host "Git:        $sha1"                             -ForegroundColor Green
$now = Get-Date -Format "yyyy'-'MM'-'dd'T'HH':'mm':'sszzz"
Write-Host "Build Date: $now"
Write-Host "" 

# Write-Output "Running go vet..."
# go vet -all ./...
# if ($LastExitCode -ne 0) {
#     exit
# }


$stdZip = ".\deploy\isitsql.$($version).zip"
If (Test-Path -Path $stdZip) {
    Write-Error "Error: $($stdZip) exists"
    Exit
}

Write-Output "Running go test..."
hottest test ./...
if ($LastExitCode -ne 0) {
    exit
}

# Write-Output "Running go generate..."
# go generate
# if ($LastExitCode -ne 0) {
#     exit
# }

Write-Output "Building isitsql.exe..."
# go build -o "$($target)\xelogstash.exe" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now" ".\cmd\xelogstash"
$path = "github.com/scalesql/isitsql/internal/build"
$flags = "-X main.buildGit=$($sha1) -X main.buildDate=$($now) -X '$path.builtFlag=$now' -X '$path.commitFlag=$sha1' -X '$path.versionFlag=$($semver.ToString())'"
go build -a -o "$($target)\isitsql.exe" -ldflags "$($flags)" -trimpath ./cmd/isitsql
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building cfg2file.exe..."
go build -a -o "$($target)\optional\cfg2file.exe" ./cmd/cfg2file
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building linter.exe..."
go build -a -o "$($target)\optional\linter.exe" ./cmd/linter
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building conntest.exe..."
go build -a -o "$($target)\optional\conntest.exe" ./cmd/conntest
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "" 
Write-Output "isitsql.exe -version..."
& "$($target)\isitsql.exe" -version
Write-Output "" 

Write-Output "Copying Files..."
go-licenses report  ./... --ignore github.com/scalesql --template static/other/notice.tmpl > deploy/isitsql/NOTICE.md
go-licenses report  ./... --ignore github.com/scalesql --template static/other/notice.tmpl > ./NOTICE.md
# Copy-Item -Path ".\README.html"    -Destination $target
Copy-Item -Path ".\LICENSE.md"   -Destination "$target\LICENSE.md"
# This is my blackfriday at https://github.com/billgraziano/blackfriday
# My blackfriday embeds the CSS in the HTML
blackfriday-tool -css .\static\docs\style.css -embed .\static\docs\README.md        ".\deploy\isitsql\README.html"
blackfriday-tool -css .\static\docs\style.css -embed .\static\docs\FileConfig.md    ".\deploy\isitsql\optional\FileConfig.html"
blackfriday-tool -css .\static\docs\style.css -embed .\static\docs\ConnTest.md      ".\deploy\isitsql\optional\ConnTest.html"

Write-Host "Writing Zip Files..."
# Standard Zip file
$stdCompress = @{
    Path = $target
    CompressionLevel = "Fastest"
    DestinationPath = $stdZip
    Update = $true
}
Compress-Archive @stdCompress

# Write-Host "Writing Dependency Zip Files..."
# # Standard Zip file
# $stdZip = ".\deploy\isitsql\dependency-licenses.zip"
# $stdCompress = @{
#     Path = ".\dependency-licenses"
#     CompressionLevel = "Fastest"
#     DestinationPath = $stdZip
#     Update = $true
# }
# Compress-Archive @stdCompress

Write-Output "Done."
