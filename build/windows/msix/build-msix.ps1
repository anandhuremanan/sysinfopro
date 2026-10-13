<#
.SYNOPSIS
  Packages the production exe as an .msix for the Microsoft Store.

.DESCRIPTION
  Build the exe first:  wails build -o sysinfopro-<version>.exe
  Then run this script with the three identity values from Partner Center
  (your app > Product management > Product identity).

  The package is left unsigned on purpose: the Store signs it during certification.
  To install it on your own PC for a test, sign it with a test certificate first.

.EXAMPLE
  .\build-msix.ps1 -IdentityName "12345Publisher.SysInfoPro" `
                   -Publisher "CN=ABCD1234-0000-0000-0000-000000000000" `
                   -PublisherDisplayName "Nada Labs"
#>
param(
    [Parameter(Mandatory = $true)][string]$IdentityName,
    [Parameter(Mandatory = $true)][string]$Publisher,
    [Parameter(Mandatory = $true)][string]$PublisherDisplayName,
    # three-part app version; defaults to productVersion in wails.json
    [string]$Version
)

$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Resolve-Path (Join-Path $here '..\..\..')

if (-not $Version) {
    $Version = (Get-Content (Join-Path $root 'wails.json') -Raw | ConvertFrom-Json).info.productVersion
}
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw "Version must look like 3.0.0, got '$Version'" }
# the Store requires four parts with the last one 0
$packageVersion = "$Version.0"

$exe = Join-Path $root "build\bin\sysinfopro-$Version.exe"
if (-not (Test-Path $exe)) { throw "Build the exe first: $exe not found" }

# newest Windows SDK that has the packaging tools
$sdkBin = Get-ChildItem 'C:\Program Files (x86)\Windows Kits\10\bin' -Directory |
    Where-Object { Test-Path (Join-Path $_.FullName 'x64\makeappx.exe') } |
    Sort-Object Name -Descending | Select-Object -First 1
if (-not $sdkBin) { throw 'Windows SDK not found (makeappx.exe). Install the Windows 10/11 SDK.' }
$makeappx = Join-Path $sdkBin.FullName 'x64\makeappx.exe'
$makepri = Join-Path $sdkBin.FullName 'x64\makepri.exe'

$stage = Join-Path $root 'build\bin\msix-stage'
if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
New-Item -ItemType Directory -Path $stage | Out-Null

Copy-Item $exe (Join-Path $stage 'sysinfopro.exe')
Copy-Item (Join-Path $here 'Assets') (Join-Path $stage 'Assets') -Recurse

$escape = { param($s) [System.Security.SecurityElement]::Escape($s) }
$manifest = (Get-Content (Join-Path $here 'AppxManifest.template.xml') -Raw).
    Replace('{{IDENTITY_NAME}}', (& $escape $IdentityName)).
    Replace('{{PUBLISHER}}', (& $escape $Publisher)).
    Replace('{{PUBLISHER_DISPLAY_NAME}}', (& $escape $PublisherDisplayName)).
    Replace('{{VERSION}}', $packageVersion)
[System.IO.File]::WriteAllText((Join-Path $stage 'AppxManifest.xml'), $manifest, (New-Object System.Text.UTF8Encoding($false)))

# resources.pri lets Windows pick the right icon size (taskbar, Start, Alt+Tab)
$priConfig = Join-Path $root 'build\bin\msix-priconfig.xml'
& $makepri createconfig /cf $priConfig /dq en-US /pv 10.0.0 /o | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'makepri createconfig failed' }
& $makepri new /pr $stage /cf $priConfig /mn (Join-Path $stage 'AppxManifest.xml') /of (Join-Path $stage 'resources.pri') /o | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'makepri new failed' }

$output = Join-Path $root "build\bin\SysInfoPro_${packageVersion}_x64.msix"
& $makeappx pack /d $stage /p $output /o | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'makeappx pack failed' }

Remove-Item $stage -Recurse -Force
Remove-Item $priConfig -Force

$sizeMB = [math]::Round((Get-Item $output).Length / 1MB, 1)
Write-Output "Package ready: $output ($sizeMB MB, version $packageVersion)"
Write-Output 'Upload it in Partner Center > Packages. It is unsigned; the Store signs it.'
