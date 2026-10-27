$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
function D($d) { if ($d) { ([datetime]$d).ToString('yyyy-MM-dd') } else { '' } }
$cs = Get-CimInstance Win32_ComputerSystem
$bios = Get-CimInstance Win32_BIOS
$board = Get-CimInstance Win32_BaseBoard
$os = Get-CimInstance Win32_OperatingSystem
# each display driver's own 64-bit memory size, by adapter name (readable without admin)
$vram = @{}
Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}' | ForEach-Object {
	$p = Get-ItemProperty -LiteralPath $_.PSPath
	$size = $p.'HardwareInformation.qwMemorySize'
	if ($p.DriverDesc -and $size) { $vram[[string]$p.DriverDesc] = [int64]$size }
}
[pscustomobject]@{
	system = [pscustomobject]@{
		manufacturer = [string]$cs.Manufacturer
		model        = [string]$cs.Model
		type         = [string]$cs.SystemType
	}
	bios = [pscustomobject]@{
		vendor  = [string]$bios.Manufacturer
		version = [string]$bios.SMBIOSBIOSVersion
		date    = D $bios.ReleaseDate
		serial  = [string]$bios.SerialNumber
	}
	board = [pscustomobject]@{
		manufacturer = [string]$board.Manufacturer
		product      = [string]$board.Product
	}
	os = [pscustomobject]@{
		name      = [string]$os.Caption
		version   = [string]$os.Version
		build     = [string]$os.BuildNumber
		arch      = [string]$os.OSArchitecture
		installed = D $os.InstallDate
		last_boot = if ($os.LastBootUpTime) { ([datetime]$os.LastBootUpTime).ToString('yyyy-MM-dd HH:mm') } else { '' }
	}
	cpus = @(Get-CimInstance Win32_Processor | ForEach-Object {
		[pscustomobject]@{
			name    = ([string]$_.Name).Trim()
			cores   = [int]$_.NumberOfCores
			threads = [int]$_.NumberOfLogicalProcessors
			max_mhz = [int]$_.MaxClockSpeed
			socket  = [string]$_.SocketDesignation
		}
	})
	gpus = @(Get-CimInstance Win32_VideoController | ForEach-Object {
		# AdapterRAM is 32-bit, so cards with more than 4 GB read as 4 GB; the
		# display driver's own 64-bit value is used when it is there
		$name = [string]$_.Name
		$vramBytes = if ($vram.ContainsKey($name)) { $vram[$name] } else { [int64]$_.AdapterRAM }
		[pscustomobject]@{
			name       = $name
			driver     = [string]$_.DriverVersion
			vram_bytes = $vramBytes
			width      = [int]$_.CurrentHorizontalResolution
			height     = [int]$_.CurrentVerticalResolution
			refresh_hz = [int]$_.CurrentRefreshRate
		}
	})
	memory = @(Get-CimInstance Win32_PhysicalMemory | ForEach-Object {
		[pscustomobject]@{
			slot         = [string]$_.DeviceLocator
			size_bytes   = [int64]$_.Capacity
			speed_mhz    = [int]$_.Speed
			manufacturer = ([string]$_.Manufacturer).Trim()
			part         = ([string]$_.PartNumber).Trim()
		}
	})
	memory_slots = [int](Get-CimInstance Win32_PhysicalMemoryArray | Measure-Object -Property MemoryDevices -Sum).Sum
	disks = @(Get-CimInstance Win32_DiskDrive | ForEach-Object {
		[pscustomobject]@{
			model      = ([string]$_.Model).Trim()
			size_bytes = [int64]$_.Size
			interface  = [string]$_.InterfaceType
			media      = [string]$_.MediaType
		}
	})
} | ConvertTo-Json -Depth 5 -Compress
