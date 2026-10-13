//go:build !windows

package functions

// IsStorePackage is only meaningful on Windows
func IsStorePackage() bool {
	return false
}
