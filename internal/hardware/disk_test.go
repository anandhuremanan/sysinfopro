package hardware

import (
	"testing"

	"sysinfopro/internal/storage"
)

func TestJudgeHealth(t *testing.T) {
	clean := &storage.NVMeHealth{PercentUsed: 13, AvailableSpare: 100, SpareThreshold: 10}
	tests := []struct {
		name       string
		windows    string
		nvme       *storage.NVMeHealth
		wantStatus string
		wantLife   int
		wantReason bool
	}{
		{"healthy NVMe", "Healthy", clean, "Healthy", 87, false},
		{"SATA drive has no health log", "Healthy", nil, "Healthy", -1, false},
		{"Windows did not answer", "", nil, "Unknown", -1, false},
		{"clean log fills in for Windows", "", clean, "Healthy", 87, false},
		{"Windows warns", "Warning", clean, "Warning", 87, true},
		{"worn out", "Healthy", &storage.NVMeHealth{PercentUsed: 104, AvailableSpare: 100, SpareThreshold: 10}, "Warning", 0, true},
		{"spare below threshold", "Healthy", &storage.NVMeHealth{PercentUsed: 40, AvailableSpare: 5, SpareThreshold: 10}, "Warning", 60, true},
		{"read-only", "Healthy", &storage.NVMeHealth{AvailableSpare: 100, SpareThreshold: 10, CriticalWarning: 0x08}, "Warning", 100, true},
		{"media errors", "Healthy", &storage.NVMeHealth{AvailableSpare: 100, SpareThreshold: 10, MediaErrors: 3}, "Warning", 100, true},
		{"Unhealthy stays Unhealthy", "Unhealthy", &storage.NVMeHealth{PercentUsed: 120, AvailableSpare: 100, SpareThreshold: 10}, "Unhealthy", 0, true},
	}
	for _, tt := range tests {
		got := judgeHealth(diskRow{Drive: "C:", Health: tt.windows}, "NVMe SSD", tt.nvme)
		if got.Status != tt.wantStatus || got.LifeLeftPct != tt.wantLife || (got.Reason != "") != tt.wantReason {
			t.Errorf("%s: got status %q, life %d, reason %q", tt.name, got.Status, got.LifeLeftPct, got.Reason)
		}
	}
}
