package functions

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

func GetMemoryStats() (*MemoryStats, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error getting memory: %v", err)
	}

	swapStat, _ := mem.SwapMemory()

	swapTotal := uint64(0)
	swapUsed := uint64(0)
	if swapStat != nil {
		swapTotal = swapStat.Total / 1024 / 1024
		swapUsed = swapStat.Used / 1024 / 1024
	}

	pressureLevel := "Optimal"
	if vmStat.UsedPercent > 85.0 {
		pressureLevel = "Critical"
	} else if vmStat.UsedPercent > 70.0 {
		pressureLevel = "Moderate"
	}

	return &MemoryStats{
		TotalRAMMB:    vmStat.Total / 1024 / 1024,
		FreeRAMMB:     vmStat.Free / 1024 / 1024,
		UsedRAMMB:     (vmStat.Total - vmStat.Free) / 1024 / 1024,
		UsagePercent:  vmStat.UsedPercent,
		Timestamp:     time.Now().Unix(),
		SwapTotalMB:   swapTotal,
		SwapUsedMB:    swapUsed,
		PressureLevel: pressureLevel,
	}, nil
}
