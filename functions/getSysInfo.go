package functions

import (
	"fmt"
	"os/user"
	"strings"

	"myproject/internal/storage"

	// "github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
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
	u, err := user.Current()
	if err != nil {
		info.LoggedInUser = "unknown"
	}
	info.LoggedInUser = u.Username

	// RAM
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error getting memory: %v", err)
	}
	info.TotalRAMMB = vmStat.Total / 1024 / 1024
	info.FreeRAMMB = vmStat.Free / 1024 / 1024

	// Disk
	partitions, err := disk.Partitions(false)
	if err != nil || len(partitions) == 0 {
		return nil, fmt.Errorf("error getting disk partitions: %v", err)
	}
	usageStat, err := disk.Usage(storage.RootPath(partitions[0].Mountpoint))
	if err != nil {
		return nil, fmt.Errorf("error getting disk usage: %v", err)
	}
	info.TotalDiskGB = usageStat.Total / 1024 / 1024 / 1024
	info.FreeDiskGB = usageStat.Free / 1024 / 1024 / 1024
	info.DiskUsagePct = usageStat.UsedPercent

	// Detect SSD vs HDD (simple check using "SSD" in device name)
	diskType := "HDD"
	if strings.Contains(strings.ToLower(partitions[0].Device), "ssd") {
		diskType = "SSD"
	}
	info.DiskType = diskType

	// Network
	netInterfaces, err := net.Interfaces()
	if err != nil || len(netInterfaces) == 0 {
		info.NetworkStatus = "disconnected"
	} else {
		connected := false
		for _, iface := range netInterfaces {
			if len(iface.Addrs) > 0 && iface.Flags[0] == "up" {
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
