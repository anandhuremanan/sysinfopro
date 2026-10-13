//go:build windows

package storage

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ioctlStorageQueryProperty = 0x002D1400

	storageDeviceProperty            = 0
	storageDeviceSeekPenaltyProperty = 7

	busTypeNVMe = 17
)

type storagePropertyQuery struct {
	PropertyID uint32
	QueryType  uint32 // 0 = PropertyStandardQuery
	Additional [1]byte
}

type deviceSeekPenaltyDescriptor struct {
	Version           uint32
	Size              uint32
	IncursSeekPenalty bool
}

// leading fields of STORAGE_DEVICE_DESCRIPTOR, up to the bus type
type storageDeviceDescriptorHead struct {
	Version               uint32
	Size                  uint32
	DeviceType            byte
	DeviceTypeModifier    byte
	RemovableMedia        bool
	CommandQueueing       bool
	VendorIDOffset        uint32
	ProductIDOffset       uint32
	ProductRevisionOffset uint32
	SerialNumberOffset    uint32
	BusType               uint32
}

// DiskType reports "NVMe SSD", "SSD" or "HDD" for the drive holding mount ("C:"),
// or "" when Windows does not say. It asks the drive whether seeking costs time,
// which works without administrator rights.
func DiskType(mount string) string {
	if len(mount) < 2 || mount[1] != ':' {
		return ""
	}
	path, err := windows.UTF16PtrFromString(`\\.\` + mount[:2])
	if err != nil {
		return ""
	}

	// access 0 is enough for property queries and needs no elevation
	handle, err := windows.CreateFile(path, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)

	var returned uint32

	seekQuery := storagePropertyQuery{PropertyID: storageDeviceSeekPenaltyProperty}
	seek := deviceSeekPenaltyDescriptor{}
	err = windows.DeviceIoControl(handle, ioctlStorageQueryProperty,
		(*byte)(unsafe.Pointer(&seekQuery)), uint32(unsafe.Sizeof(seekQuery)),
		(*byte)(unsafe.Pointer(&seek)), uint32(unsafe.Sizeof(seek)), &returned, nil)
	if err != nil {
		return ""
	}
	if seek.IncursSeekPenalty {
		return "HDD"
	}

	deviceQuery := storagePropertyQuery{PropertyID: storageDeviceProperty}
	device := storageDeviceDescriptorHead{}
	err = windows.DeviceIoControl(handle, ioctlStorageQueryProperty,
		(*byte)(unsafe.Pointer(&deviceQuery)), uint32(unsafe.Sizeof(deviceQuery)),
		(*byte)(unsafe.Pointer(&device)), uint32(unsafe.Sizeof(device)), &returned, nil)
	if err == nil && device.BusType == busTypeNVMe {
		return "NVMe SSD"
	}
	return "SSD"
}
