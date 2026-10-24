//go:build windows

package storage

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	storageDeviceProtocolSpecificProperty = 50
	protocolTypeNvme                      = 3
	nvmeDataTypeLogPage                   = 2
	nvmeLogPageHealthInfo                 = 2
	nvmeHealthLogSize                     = 512
)

// STORAGE_PROTOCOL_SPECIFIC_DATA
type storageProtocolSpecificData struct {
	ProtocolType                 uint32
	DataType                     uint32
	ProtocolDataRequestValue     uint32
	ProtocolDataRequestSubValue  uint32
	ProtocolDataOffset           uint32
	ProtocolDataLength           uint32
	FixedProtocolReturnData      uint32
	ProtocolDataRequestSubValue2 uint32
	ProtocolDataRequestSubValue3 uint32
	ProtocolDataRequestSubValue4 uint32
}

// ReadNVMeHealth reads the NVMe health log of the drive holding mount ("C:").
// It uses a handle opened without read or write access, which Windows allows
// for property queries without administrator rights.
func ReadNVMeHealth(mount string) (*NVMeHealth, error) {
	if len(mount) < 2 || mount[1] != ':' {
		return nil, fmt.Errorf("not a drive letter: %q", mount)
	}
	path, err := windows.UTF16PtrFromString(`\\.\` + mount[:2])
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(path, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	const queryHead = 8 // PropertyId + QueryType
	protoSize := int(unsafe.Sizeof(storageProtocolSpecificData{}))
	buf := make([]byte, queryHead+protoSize+nvmeHealthLogSize)

	binary.LittleEndian.PutUint32(buf[0:], storageDeviceProtocolSpecificProperty)
	binary.LittleEndian.PutUint32(buf[4:], 0) // PropertyStandardQuery
	proto := (*storageProtocolSpecificData)(unsafe.Pointer(&buf[queryHead]))
	proto.ProtocolType = protocolTypeNvme
	proto.DataType = nvmeDataTypeLogPage
	proto.ProtocolDataRequestValue = nvmeLogPageHealthInfo
	proto.ProtocolDataOffset = uint32(protoSize)
	proto.ProtocolDataLength = nvmeHealthLogSize

	var returned uint32
	err = windows.DeviceIoControl(handle, ioctlStorageQueryProperty,
		&buf[0], uint32(len(buf)), &buf[0], uint32(len(buf)), &returned, nil)
	if err != nil {
		return nil, err
	}

	// the reply is STORAGE_PROTOCOL_DATA_DESCRIPTOR: Version, Size, then the same
	// protocol block, whose offset is relative to that block
	reply := (*storageProtocolSpecificData)(unsafe.Pointer(&buf[queryHead]))
	start := queryHead + int(reply.ProtocolDataOffset)
	if reply.ProtocolDataLength < nvmeHealthLogSize || start+nvmeHealthLogSize > len(buf) {
		return nil, fmt.Errorf("drive returned a short health log")
	}
	return parseNVMeHealthLog(buf[start : start+nvmeHealthLogSize]), nil
}
