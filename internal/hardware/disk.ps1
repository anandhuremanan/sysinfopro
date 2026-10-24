# One line per drive letter: which physical disk holds it and what Windows says about
# that disk's health. Read-only; works without administrator rights.
$ErrorActionPreference = 'SilentlyContinue'
$disks = @{}
foreach ($pd in Get-PhysicalDisk) { $disks[[string]$pd.DeviceId] = $pd }
$rows = foreach ($p in Get-Partition | Where-Object DriveLetter) {
  $pd = $disks[[string]$p.DiskNumber]
  [pscustomobject]@{
    drive  = "$($p.DriveLetter):"
    model  = [string]$pd.FriendlyName
    health = [string]$pd.HealthStatus
  }
}
ConvertTo-Json -InputObject @($rows) -Compress
