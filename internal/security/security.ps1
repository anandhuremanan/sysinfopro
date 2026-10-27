$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
function Reg($path, $name) {
	$v = Get-ItemProperty -Path $path -Name $name
	if ($null -ne $v) { $v.$name } else { $null }
}
$mp = Get-MpComputerStatus
$bl = $null
try { $bl = (New-Object -ComObject Shell.Application).NameSpace($env:SystemDrive).Self.ExtendedProperty('System.Volume.BitLockerProtection') } catch {}
$hotfix = Get-HotFix | Where-Object { $_.InstalledOn } | Sort-Object InstalledOn -Descending | Select-Object -First 1
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
[pscustomobject]@{
	antivirus = @(Get-CimInstance -Namespace root/SecurityCenter2 -ClassName AntiVirusProduct | ForEach-Object {
		[pscustomobject]@{ name = [string]$_.displayName; state = [int64]$_.productState }
	})
	defender = [pscustomobject]@{
		available      = [bool]($null -ne $mp)
		realtime       = [bool]$mp.RealTimeProtectionEnabled
		signature_days = if ($null -ne $mp -and $mp.AntivirusSignatureLastUpdated) { [int]((Get-Date) - $mp.AntivirusSignatureLastUpdated).TotalDays } else { -1 }
	}
	firewall = @(Get-NetFirewallProfile -PolicyStore ActiveStore | ForEach-Object {
		[pscustomobject]@{ name = [string]$_.Name; enabled = [bool]($_.Enabled -eq 'True' -or $_.Enabled -eq 1) }
	})
	uac          = Reg 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' 'EnableLUA'
	secure_boot  = Reg 'HKLM:\SYSTEM\CurrentControlSet\Control\SecureBoot\State' 'UEFISecureBootEnabled'
	bitlocker    = $bl
	reboot_pending = [bool]((Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending') -or (Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired'))
	last_update_days = if ($hotfix) { [int]((Get-Date) - $hotfix.InstalledOn).TotalDays } else { -1 }
	is_admin     = [bool](New-Object Security.Principal.WindowsPrincipal($identity)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
	rdp_deny     = Reg 'HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server' 'fDenyTSConnections'
} | ConvertTo-Json -Depth 4 -Compress
