$ErrorActionPreference = "Stop"
$sha1 = (git describe --tags --dirty --always).Trim()
# Write-Output "Git: $sha1"
$now = Get-Date -Format "yyyy'-'MM'-'dd'T'HH':'mm':'sszzz"
$path = "github.com/scalesql/isitsql/internal/build"
$flags = @()
$flags += "-X main.buildGit=$($sha1)"
$flags += "-X main.buildDate=$($now)"
$flags += "-X '$path.builtFlag=$now'"
$flags += "-X '$path.commitFlag=$sha1'" 
$flags += "-X '$path.versionFlag=0.1'"
# $flags += "-trimpath=C:\\dev\\github.com\\isitsql"
$flagParameters = $flags -Join " "
Write-Output "Building..."
go build -tags dev -ldflags "$($flagParameters)" -trimpath ./cmd/isitsql
if ($LastExitCode -ne 0) {
    exit
}
Write-Output "----------------------------------------------------"
.\isitsql.exe -debug
Write-Output "LastExitCode: $($LastExitCode)"
