package functions

import "testing"

func TestIsStorePackageFalseForPlainExe(t *testing.T) {
	if IsStorePackage() {
		t.Error("a test binary run from disk has no package identity")
	}
}
