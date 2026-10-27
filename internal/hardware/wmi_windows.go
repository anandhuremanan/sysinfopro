//go:build windows

package hardware

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

// The inventory and drive health are read straight from WMI, in this process.
// Starting PowerShell for the same queries cost 1-2 seconds and 60-90 MB each
// time, and security software often watches or blocks PowerShell.

// withDeadline runs fn, giving up when ctx ends; a hung WMI provider then leaves
// a goroutine waiting, but never the caller
func withDeadline[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := fn()
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		return r.v, r.err
	case <-ctx.Done():
		var zero T
		//lint:ignore ST1005 shown to the user as is; "Windows" is a name
		return zero, fmt.Errorf("Windows did not answer in time: %w", ctx.Err())
	}
}

// --- drive health (Get-PhysicalDisk / Get-Partition) ---

const storageNamespace = `root\Microsoft\Windows\Storage`

type msftPhysicalDisk struct {
	DeviceId     string
	FriendlyName string
	HealthStatus uint16 // 0 healthy, 1 warning, 2 unhealthy, 5 unknown
}

type msftPartition struct {
	DriveLetter uint16 // a character code; 0 when the partition has no letter
	DiskNumber  uint32
}

func healthWord(status uint16) string {
	switch status {
	case 0:
		return "Healthy"
	case 1:
		return "Warning"
	case 2:
		return "Unhealthy"
	}
	return "Unknown"
}

// readDiskRows lists each drive letter with the physical disk that holds it
func readDiskRows(ctx context.Context) ([]diskRow, error) {
	return withDeadline(ctx, func() ([]diskRow, error) {
		var disks []msftPhysicalDisk
		if err := wmi.QueryNamespace("SELECT DeviceId, FriendlyName, HealthStatus FROM MSFT_PhysicalDisk", &disks, storageNamespace); err != nil {
			return nil, fmt.Errorf("could not list physical disks: %w", err)
		}
		var parts []msftPartition
		if err := wmi.QueryNamespace("SELECT DriveLetter, DiskNumber FROM MSFT_Partition", &parts, storageNamespace); err != nil {
			return nil, fmt.Errorf("could not list partitions: %w", err)
		}
		byID := map[string]msftPhysicalDisk{}
		for _, d := range disks {
			byID[d.DeviceId] = d
		}
		rows := []diskRow{}
		for _, p := range parts {
			if p.DriveLetter == 0 {
				continue
			}
			d := byID[fmt.Sprint(p.DiskNumber)]
			rows = append(rows, diskRow{
				Drive:  string(rune(p.DriveLetter)) + ":",
				Model:  d.FriendlyName,
				Health: healthWord(d.HealthStatus),
			})
		}
		return rows, nil
	})
}

// --- hardware inventory ---

type win32ComputerSystem struct {
	Manufacturer string
	Model        string
	SystemType   string
}

type win32BIOS struct {
	Manufacturer      string
	SMBIOSBIOSVersion string
	ReleaseDate       *time.Time
	SerialNumber      string
}

type win32BaseBoard struct {
	Manufacturer string
	Product      string
}

type win32OperatingSystem struct {
	Caption        string
	Version        string
	BuildNumber    string
	OSArchitecture string
	InstallDate    *time.Time
	LastBootUpTime *time.Time
}

type win32Processor struct {
	Name                      string
	NumberOfCores             uint32
	NumberOfLogicalProcessors uint32
	MaxClockSpeed             uint32
	SocketDesignation         string
}

type win32VideoController struct {
	Name                        string
	DriverVersion               string
	AdapterRAM                  *uint32
	CurrentHorizontalResolution *uint32
	CurrentVerticalResolution   *uint32
	CurrentRefreshRate          *uint32
}

type win32PhysicalMemory struct {
	DeviceLocator string
	Capacity      uint64
	Speed         *uint32
	Manufacturer  string
	PartNumber    string
}

type win32PhysicalMemoryArray struct {
	MemoryDevices *uint32
}

type win32DiskDrive struct {
	Model         string
	Size          *uint64
	InterfaceType string
	MediaType     string
}

func day(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02")
}

func u32(p *uint32) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

// one query into dst; "" when the class could not be read is left to the caller
func query[T any](q string) ([]T, error) {
	var dst []T
	err := wmi.Query(q, &dst)
	return dst, err
}

// readSpecs reads the inventory. A class that cannot be read leaves its part
// empty rather than failing the whole inventory, as the PowerShell version did.
func readSpecs(ctx context.Context) (*HardwareSpecs, error) {
	return withDeadline(ctx, func() (*HardwareSpecs, error) {
		s := &HardwareSpecs{CPUs: []CPUSpec{}, GPUs: []GPUSpec{}, Memory: []MemoryModuleSpec{}, Disks: []DiskSpec{}}
		var firstErr error
		note := func(err error) {
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}

		cs, err := query[win32ComputerSystem]("SELECT Manufacturer, Model, SystemType FROM Win32_ComputerSystem")
		note(err)
		if len(cs) > 0 {
			s.System = SystemSpec{Manufacturer: cs[0].Manufacturer, Model: cs[0].Model, Type: cs[0].SystemType}
		}
		bios, err := query[win32BIOS]("SELECT Manufacturer, SMBIOSBIOSVersion, ReleaseDate, SerialNumber FROM Win32_BIOS")
		note(err)
		if len(bios) > 0 {
			b := bios[0]
			s.BIOS = BIOSSpec{Vendor: b.Manufacturer, Version: b.SMBIOSBIOSVersion, Date: day(b.ReleaseDate), Serial: b.SerialNumber}
		}
		board, err := query[win32BaseBoard]("SELECT Manufacturer, Product FROM Win32_BaseBoard")
		note(err)
		if len(board) > 0 {
			s.Board = BoardSpec{Manufacturer: board[0].Manufacturer, Product: board[0].Product}
		}
		osInfo, err := query[win32OperatingSystem]("SELECT Caption, Version, BuildNumber, OSArchitecture, InstallDate, LastBootUpTime FROM Win32_OperatingSystem")
		note(err)
		if len(osInfo) > 0 {
			o := osInfo[0]
			s.OS = OSSpec{Name: o.Caption, Version: o.Version, Build: o.BuildNumber, Arch: o.OSArchitecture, Installed: day(o.InstallDate)}
			if o.LastBootUpTime != nil && !o.LastBootUpTime.IsZero() {
				s.OS.LastBoot = o.LastBootUpTime.Local().Format("2006-01-02 15:04")
			}
		}
		cpus, err := query[win32Processor]("SELECT Name, NumberOfCores, NumberOfLogicalProcessors, MaxClockSpeed, SocketDesignation FROM Win32_Processor")
		note(err)
		for _, c := range cpus {
			s.CPUs = append(s.CPUs, CPUSpec{Name: strings.TrimSpace(c.Name), Cores: int(c.NumberOfCores), Threads: int(c.NumberOfLogicalProcessors), MaxMHz: int(c.MaxClockSpeed), Socket: c.SocketDesignation})
		}
		vram := driverVideoMemory()
		gpus, err := query[win32VideoController]("SELECT Name, DriverVersion, AdapterRAM, CurrentHorizontalResolution, CurrentVerticalResolution, CurrentRefreshRate FROM Win32_VideoController")
		note(err)
		for _, g := range gpus {
			// AdapterRAM is 32-bit, so cards with more than 4 GB read as 4 GB; the
			// display driver's own 64-bit value is used when it is there
			bytes := int64(u32(g.AdapterRAM))
			if v, ok := vram[g.Name]; ok {
				bytes = v
			}
			s.GPUs = append(s.GPUs, GPUSpec{Name: g.Name, Driver: g.DriverVersion, VRAMBytes: bytes,
				Width: u32(g.CurrentHorizontalResolution), Height: u32(g.CurrentVerticalResolution), RefreshHz: u32(g.CurrentRefreshRate)})
		}
		mem, err := query[win32PhysicalMemory]("SELECT DeviceLocator, Capacity, Speed, Manufacturer, PartNumber FROM Win32_PhysicalMemory")
		note(err)
		for _, m := range mem {
			s.Memory = append(s.Memory, MemoryModuleSpec{Slot: m.DeviceLocator, SizeBytes: int64(m.Capacity), SpeedMHz: u32(m.Speed),
				Manufacturer: strings.TrimSpace(m.Manufacturer), Part: strings.TrimSpace(m.PartNumber)})
		}
		arrays, err := query[win32PhysicalMemoryArray]("SELECT MemoryDevices FROM Win32_PhysicalMemoryArray")
		note(err)
		for _, a := range arrays {
			s.MemorySlots += u32(a.MemoryDevices)
		}
		disks, err := query[win32DiskDrive]("SELECT Model, Size, InterfaceType, MediaType FROM Win32_DiskDrive")
		note(err)
		for _, d := range disks {
			size := int64(0)
			if d.Size != nil {
				size = int64(*d.Size)
			}
			s.Disks = append(s.Disks, DiskSpec{Model: strings.TrimSpace(d.Model), SizeBytes: size, Interface: d.InterfaceType, Media: d.MediaType})
		}

		// nothing at all could be read: report it instead of an empty inventory
		if firstErr != nil && len(cs) == 0 && len(cpus) == 0 && len(osInfo) == 0 {
			return nil, fmt.Errorf("could not read hardware inventory: %w", firstErr)
		}
		return s, nil
	})
}

// driverVideoMemory reads each display driver's 64-bit memory size by adapter
// name, from the registry (readable without administrator rights)
func driverVideoMemory() map[string]int64 {
	out := map[string]int64{}
	const class = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, class, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return out
	}
	defer k.Close()
	names, _ := k.ReadSubKeyNames(-1)
	for _, n := range names {
		sub, err := registry.OpenKey(k, n, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		desc, _, _ := sub.GetStringValue("DriverDesc")
		size := memorySize(sub)
		sub.Close()
		if desc != "" && size > 0 {
			out[desc] = size
		}
	}
	return out
}

// memorySize reads HardwareInformation.qwMemorySize, stored as a QWORD or as 8 raw bytes
func memorySize(k registry.Key) int64 {
	const name = "HardwareInformation.qwMemorySize"
	if v, _, err := k.GetIntegerValue(name); err == nil {
		return int64(v)
	}
	if b, _, err := k.GetBinaryValue(name); err == nil && len(b) >= 8 {
		return int64(binary.LittleEndian.Uint64(b[:8]))
	}
	return 0
}
