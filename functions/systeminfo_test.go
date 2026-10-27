//go:build integration

package functions

import (
	"testing"
)

func TestGetSystemInfo(t *testing.T) {
	info, err := GetSystemInfo()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if info.CPUModel == "" {
		t.Errorf("Expected CPUModel to be set, got empty string")
	}
	if info.TotalRAMMB == 0 {
		t.Errorf("Expected TotalRAMMB > 0, got 0")
	}
	if info.LoggedInUser == "" || info.LoggedInUser == "unknown" {
		t.Errorf("Expected LoggedInUser to be set, got empty or unknown")
	}
}
