//go:build windows

package vdisk

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const lxssKey = `Software\Microsoft\Windows\CurrentVersion\Lxss`

// registeredDistros reads WSL's own list of distributions for the current user
func registeredDistros() []distro {
	root, err := registry.OpenKey(registry.CURRENT_USER, lxssKey, registry.READ)
	if err != nil {
		return nil // WSL has never been used
	}
	defer root.Close()

	defaultID, _, _ := root.GetStringValue("DefaultDistribution")
	ids, _ := root.ReadSubKeyNames(-1)

	var distros []distro
	for _, id := range ids {
		key, err := registry.OpenKey(root, id, registry.READ)
		if err != nil {
			continue
		}
		name, _, _ := key.GetStringValue("DistributionName")
		base, _, _ := key.GetStringValue("BasePath")
		vhd, _, err := key.GetStringValue("VhdFileName")
		if err != nil || vhd == "" {
			vhd = "ext4.vhdx"
		}
		version, _, err := key.GetIntegerValue("Version")
		if err != nil {
			version = 2
		}
		key.Close()
		if name == "" || base == "" {
			continue
		}
		distros = append(distros, distro{
			name:      name,
			basePath:  strings.TrimPrefix(base, `\\?\`),
			vhdName:   vhd,
			version:   int(version),
			isDefault: strings.EqualFold(id, defaultID),
		})
	}
	return distros
}

var procGetCompressedFileSize = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCompressedFileSizeW")

// allocated returns the space a file really takes on disk, which is less than
// its size when it is sparse or compressed, and whether it is sparse
func allocated(path string, size int64) (int64, bool) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return size, false
	}
	sparse := false
	if attrs, err := windows.GetFileAttributes(p); err == nil {
		sparse = attrs&windows.FILE_ATTRIBUTE_SPARSE_FILE != 0
	}
	var high uint32
	low, _, callErr := procGetCompressedFileSize.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&high)))
	if uint32(low) == 0xFFFFFFFF && callErr != windows.ERROR_SUCCESS {
		return size, sparse
	}
	return int64(high)<<32 | int64(uint32(low)), sparse
}
