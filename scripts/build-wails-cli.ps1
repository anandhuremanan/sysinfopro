<#
.SYNOPSIS
  Builds the Wails CLI this project needs and installs it as wails-go127.

.DESCRIPTION
  The stock Wails v2.17.0 CLI cannot read Go 1.26+ source (its bundled
  golang.org/x/tools is too old to parse the new syntax), so `wails build`
  fails on this project. This script builds the same CLI version with a newer
  golang.org/x/tools, in a throwaway module, and installs it next to the Go
  binaries as wails-go127.exe, so it never replaces a stock `wails` install.

  Run it once per machine (and in CI). It is safe to re-run.

.EXAMPLE
  .\scripts\build-wails-cli.ps1
  wails-go127 build -o sysinfopro-3.2.0.exe
#>
param(
    [string]$WailsVersion = 'v2.17.0',
    [string]$XToolsVersion = 'v0.52.0',
    [string]$Name = 'wails-go127',
    # where to put the exe; defaults to $(go env GOPATH)\bin
    [string]$OutDir
)

$ErrorActionPreference = 'Stop'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go is not on PATH. Install the Go version in go.mod first.' }
if (-not $OutDir) { $OutDir = Join-Path (go env GOPATH) 'bin' }
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$out = Join-Path $OutDir "$Name.exe"

$work = Join-Path ([System.IO.Path]::GetTempPath()) ("wails-cli-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $work | Out-Null
Push-Location $work
# go prints progress on stderr; Windows PowerShell 5.1 would treat that as a
# failure under 'Stop', so rely on exit codes for the native commands instead
$ErrorActionPreference = 'Continue'
try {
    # a throwaway module that pins Wails and the newer x/tools together
    go mod init wailscli | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'go mod init failed' }
    go get "github.com/wailsapp/wails/v2/cmd/wails@$WailsVersion"
    if ($LASTEXITCODE -ne 0) { throw "go get wails $WailsVersion failed" }
    go get "golang.org/x/tools@$XToolsVersion"
    if ($LASTEXITCODE -ne 0) { throw "go get x/tools $XToolsVersion failed" }
    # no `go mod tidy`: this module has no source of its own, so tidy would drop the pins
    go build -trimpath -o $out github.com/wailsapp/wails/v2/cmd/wails
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
}
finally {
    Pop-Location
    Remove-Item $work -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Output "Installed $out (Wails $WailsVersion, golang.org/x/tools $XToolsVersion)"
& $out version
