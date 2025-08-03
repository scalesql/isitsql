# Advanced IsItSQL Topics
* [Deploy New Releases Using PowerShell](#deploy)
* [Send Missing Backup Alerts Using PowerShell](#backup-alerts)


<a id="deploy"></a>
## Deploy New Releases Using PowerShell
This script deploys a new executable from the ZIP file to IsItSQL running as a service on a Windows box.  From a PowerShell Administrative session, call the script like `.\Deploy 2.5-beta.3`. The script

1. Renames the existing executable with a timestamp.  This doesn't impact the running service.
2. Copies the executable to the current folder
3. Restarts the service
4. Prints the first 20 lines of the log file.

```powershell
param (
    [string] $Zip 
)

$ErrorActionPreference = "Stop"
# Load required .NET assemblies. Not necessary on PS Core 7+.
Add-Type -Assembly System.IO.Compression.FileSystem

$archivePath = "IsItSQL.$($Zip).zip"
$destinationDir = '.'
$fileName = "isitsql.exe"
$ts = (Get-Date).ToString("yyyyMMdd_HHmmss")

# Relative path of file in ZIP to extract.
# Use FORWARD slashes as directory separator, e. g. 'subdir/test.txt'
$fileToExtract = "isitsql\$($fileName)"

# Create destination dir if not exist.
# $null = New-Item $destinationDir -ItemType Directory -Force

# Convert (possibly relative) paths for safe use with .NET APIs
$resolvedArchivePath    = Convert-Path -LiteralPath $archivePath
$resolvedDestinationDir = Convert-Path -LiteralPath $destinationDir
$resolvedTarget         = Convert-Path -LiteralPath $fileName

Write-Host "Zip File: $($resolvedArchivePath)"
Write-Host "Path: $($resolvedDestinationDir)" 
Write-Host "Target: $($resolvedTarget)" 

# If the archive doesn't exist, exit
If ([System.IO.File]::Exists($resolvedArchivePath) -eq $False) {
    Write-Host "Not Found: $($Zip)" -ForegroundColor Red
    Exit 
}

# Rename the file 
$file = Get-Item $resolvedTarget
$newFileName = "$($file.BaseName)_$($ts)$($file.Extension)"
If ([System.IO.File]::Exists($resolvedTarget) -eq $True) {
    Write-Host "Renaming $($resolvedTarget) => $($newFileName)..."
    Rename-Item -Path $resolvedTarget -NewName $newFileName
}

$archive = [IO.Compression.ZipFile]::OpenRead( $resolvedArchivePath )


try {
    # Locate the desired file in the ZIP archive.
    # Replace $_.Fullname by $_.Name if file shall be found in any sub directory.
    if( $foundFile = $archive.Entries.Where({ $_.FullName -eq $fileToExtract }, 'First') ) {
    
        # Combine destination dir path and name of file in ZIP
        $destinationFile = Join-Path $resolvedDestinationDir $foundFile.Name
        Write-Host "Extracting $($fileToExtract) to $($destinationFile)..."

        # Extract the file.
        [IO.Compression.ZipFileExtensions]::ExtractToFile( $foundFile[ 0 ], $destinationFile )
    }
    else {
        Write-Host "File not found in ZIP: $fileToExtract" -ForegroundColor Red
    }
}
finally {
    # Dispose the archive so the file will be unlocked again.
    if( $archive ) {
        $archive.Dispose()
    }
}

Write-Host "Restarting IsItSQL..."
Restart-Service -Name IsItSQL

# Dump the log file
Write-Host "Dumping Log..."
Write-Host "--------------------------------------------------------------------"
Start-Sleep -Seconds 1 
$lastLog = Get-ChildItem ".\log" | Select-Object -Last 1
# $lastLog
$lastLogQualified = ".\log\$($lastLog.Name)"
Get-Content $lastLogQualified -Head 20 
Write-Host "--------------------------------------------------------------------"
Write-Host "Done."
```




<a id="backup-alerts"></a>
## Send Missing Backup Alerts Using PowerShell

```powershell
$to =   "name1 <mail1@domain.com>", "name2 <mail2@domain.com>"
$from = "name <mail3@domain.com>"
$smtp = "smtp.domain.com"
$url =  "http://localhost:8143/backups/json"

$post = "
    <p>/server/path/to/the/job</p>
    <p>Generated at 8:45am, 12:30pm, and 4:15pm by a job.</p>
"
Write-Output "Getting missing backups..."

$a = Invoke-RestMethod -Uri $url
$obj = @()
$obj += $a | 
    SELECT -Property ServerName, 
        @{Name="DBs";Expression={$_.Count }}, 
        @{Name="Last Full";Expression={"{0:dd MMM yyy HH:mm:ss GMT}" -f $_.OldestBackup }},
        @{Name="Last Log";Expression={"{0:dd MMM yyy HH:mm:ss GMT}" -f $_.OldestLogBackup }} |
    Sort-Object -Property ServerName 
$count = [int]$obj.Length 
Write-Output "Missing Backups: $($count)"

# Exit if length is zero
if ($count -eq 0) {
    Write-Output "Done."
    exit
}

$style = "
    <style>
    table, th, td {
       border: 1px solid black;
       padding: 2px;
    }
    </style>
"

$tbl = $obj | ConvertTo-Html  -Title Missing -Head $style -PreContent "<h1>Missing Backups</h1>" -PostContent $post
$html = $tbl | Out-String

Write-Output "Sending..."
Send-MailMessage -SmtpServer $smtp -Subject "Missing Database Backups" -Body $html -From $from -To $to -BodyAsHtml

Write-Output "Done."
```