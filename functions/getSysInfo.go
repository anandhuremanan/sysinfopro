package functions

import (
	"fmt"
	stdnet "net"
	"os/user"

	"sysinfopro/internal/storage"

	// "github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// CPU Info
	cpuInfo, err := cpu.Info()
	if err != nil || len(cpuInfo) == 0 {
		return nil, fmt.Errorf("error getting CPU info: %v", err)
	}
	info.CPUModel = cpuInfo[0].ModelName
	info.Manufacturer = cpuInfo[0].VendorID
	info.Cores = int(cpuInfo[0].Cores)

	//OS
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting OS info: %v", err)
	}

	info.OS = hostInfo.OS
	info.Platform = hostInfo.Platform
	info.PlatformVersion = hostInfo.PlatformVersion

	// User
	info.LoggedInUser = "unknown"
	if u, err := user.Current(); err == nil {
		info.LoggedInUser = u.Username
	}

	// RAM
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error getting memory: %v", err)
	}
	info.TotalRAMMB = vmStat.Total / 1024 / 1024
	info.FreeRAMMB = vmStat.Free / 1024 / 1024

	// Disk: the drive Windows is on (listed first), not whichever letter comes first
	drives, err := storage.ListDrives()
	if err != nil || len(drives) == 0 {
		return nil, fmt.Errorf("error getting drives: %v", err)
	}
	system := drives[0]
	info.TotalDiskGB = system.TotalBytes / 1024 / 1024 / 1024
	info.FreeDiskGB = system.FreeBytes / 1024 / 1024 / 1024
	info.TotalDiskBytes = system.TotalBytes
	info.FreeDiskBytes = system.FreeBytes
	info.DiskUsagePct = system.UsedPct

	// SSD vs HDD as reported by the drive itself; empty when unknown
	info.DiskType = storage.DiskType(system.Path)

	// Network
	netInterfaces, err := net.Interfaces()
	if err != nil || len(netInterfaces) == 0 {
		info.NetworkStatus = "disconnected"
	} else {
		connected := false
		for _, iface := range netInterfaces {
			if hasRoutableAddress(iface) {
				connected = true
				break
			}
		}
		if connected {
			info.NetworkStatus = "connected"
		} else {
			info.NetworkStatus = "disconnected"
		}
	}

	return info, nil
}

// hasRoutableAddress is true for an adapter that is up, is not the loopback
// adapter (which every PC has, connected or not), and has an address that is
// not self-assigned (169.254.x.x / fe80::), which Windows gives when no network answered
func hasRoutableAddress(iface net.InterfaceStat) bool {
	up, loopback := false, false
	for _, f := range iface.Flags {
		switch f {
		case "up":
			up = true
		case "loopback":
			loopback = true
		}
	}
	if !up || loopback {
		return false
	}
	for _, a := range iface.Addrs {
		ip, _, err := stdnet.ParseCIDR(a.Addr)
		if err != nil {
			ip = stdnet.ParseIP(a.Addr)
		}
		if ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified() {
			return true
		}
	}
	return false
}
