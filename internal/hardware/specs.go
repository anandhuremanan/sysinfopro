package hardware

import (
	"context"
	_ "embed"
	"time"
)

type SystemSpec struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Type         string `json:"type"`
}

type BIOSSpec struct {
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
	Date    string `json:"date"`
	Serial  string `json:"serial"`
}

type BoardSpec struct {
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
}

type OSSpec struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Build     string `json:"build"`
	Arch      string `json:"arch"`
	Installed string `json:"installed"`
	LastBoot  string `json:"last_boot"`
}

type CPUSpec struct {
	Name    string `json:"name"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
	MaxMHz  int    `json:"max_mhz"`
	Socket  string `json:"socket"`
}

type GPUSpec struct {
	Name      string `json:"name"`
	Driver    string `json:"driver"`
	VRAMBytes int64  `json:"vram_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	RefreshHz int    `json:"refresh_hz"`
}

type MemoryModuleSpec struct {
	Slot         string `json:"slot"`
	SizeBytes    int64  `json:"size_bytes"`
	SpeedMHz     int    `json:"speed_mhz"`
	Manufacturer string `json:"manufacturer"`
	Part         string `json:"part"`
}

type DiskSpec struct {
	Model     string `json:"model"`
	SizeBytes int64  `json:"size_bytes"`
	Interface string `json:"interface"`
	Media     string `json:"media"`
}

type HardwareSpecs struct {
	System      SystemSpec         `json:"system"`
	BIOS        BIOSSpec           `json:"bios"`
	Board       BoardSpec          `json:"board"`
	OS          OSSpec             `json:"os"`
	CPUs        []CPUSpec          `json:"cpus"`
	GPUs        []GPUSpec          `json:"gpus"`
	Memory      []MemoryModuleSpec `json:"memory"`
	MemorySlots int                `json:"memory_slots"`
	Disks       []DiskSpec         `json:"disks"`
}

// GetHardwareSpecs reads the machine's hardware inventory from Windows (WMI)
func GetHardwareSpecs(ctx context.Context) (*HardwareSpecs, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return readSpecs(ctx)
}
