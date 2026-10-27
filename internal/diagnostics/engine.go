package diagnostics

import (
	"fmt"
	"math"
)

type FactorScore struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`  // 0 - 100
	Status  string  `json:"status"` // Optimal, Fair, Warning, Critical
	Details string  `json:"details"`
}

type Recommendation struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"` // info, warning, critical
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionLabel string `json:"action_label,omitempty"`
}

type DiagnosticReport struct {
	OverallScore    float64          `json:"overall_score"`
	StatusText      string           `json:"status_text"`
	Factors         []FactorScore    `json:"factors"`
	Recommendations []Recommendation `json:"recommendations"`
}

type EvaluationInput struct {
	CpuUsagePct        float64
	IsThrottling       bool
	RamUsagePct        float64
	SwapUsedMB         uint64
	SwapTotalMB        uint64
	FreeDiskGB         uint64
	TotalDiskGB        uint64
	BatteryPct         float64
	BatteryDegradation float64
	NetworkConnected   bool
	PingLatencyMs      float64
}

func EvaluateSystemHealth(input EvaluationInput) DiagnosticReport {
	factors := []FactorScore{}
	recs := []Recommendation{}

	// 1. CPU Factor
	cpuScore := 100.0 - (input.CpuUsagePct * 0.5)
	if input.IsThrottling {
		cpuScore -= 20.0
	}
	cpuScore = math.Max(0.0, math.Min(100.0, cpuScore))
	cpuStatus := "Optimal"
	if cpuScore < 50 {
		cpuStatus = "Critical"
		recs = append(recs, Recommendation{
			ID:          "cpu_heavy",
			Severity:    "critical",
			Title:       "High CPU Utilization",
			Description: "Your CPU is under sustained heavy load or thermal pressure.",
			ActionLabel: "Check Top Processes",
		})
	} else if cpuScore < 75 {
		cpuStatus = "Fair"
	}
	factors = append(factors, FactorScore{
		Name:    "CPU Performance",
		Score:   round2(cpuScore),
		Status:  cpuStatus,
		Details: fmt.Sprintf("%.1f%% Load", input.CpuUsagePct),
	})

	// 2. RAM & Swap Factor
	ramScore := 100.0 - input.RamUsagePct
	swapPct := 0.0
	if input.SwapTotalMB > 0 && input.SwapUsedMB > 0 {
		swapPct = (float64(input.SwapUsedMB) / float64(input.SwapTotalMB)) * 100.0
	}
	// 90% is also where the home screen's memory bar turns red, so the two agree
	if swapPct > 50 {
		ramScore -= 15.0
	}
	if swapPct > 50 || input.RamUsagePct >= 90 {
		recs = append(recs, Recommendation{
			ID:          "ram_swap",
			Severity:    "warning",
			Title:       "Memory is nearly full",
			Description: "When memory runs out, Windows moves things to the drive, which is much slower.",
			ActionLabel: "Close Heavy Apps",
		})
	}
	ramScore = math.Max(0.0, math.Min(100.0, ramScore))
	ramStatus := "Optimal"
	if ramScore < 50 {
		ramStatus = "Critical"
	} else if ramScore < 75 {
		ramStatus = "Fair"
	}
	factors = append(factors, FactorScore{
		Name:    "Memory Capacity",
		Score:   round2(ramScore),
		Status:  ramStatus,
		Details: fmt.Sprintf("%.1f%% RAM Used", input.RamUsagePct),
	})

	// 3. Disk Factor
	diskScore := 100.0
	if input.TotalDiskGB > 0 {
		freePct := (float64(input.FreeDiskGB) / float64(input.TotalDiskGB)) * 100.0
		diskScore = freePct
		if freePct < 15.0 {
			recs = append(recs, Recommendation{
				ID:          "disk_low",
				Severity:    "warning",
				Title:       "Low Disk Storage",
				Description: fmt.Sprintf("Less than 15%% storage remaining (%.1f GB free).", float64(input.FreeDiskGB)),
				ActionLabel: "Run Disk Cleaner",
			})
		}
	}
	diskScore = math.Max(0.0, math.Min(100.0, diskScore))
	diskStatus := "Optimal"
	if diskScore < 30 {
		diskStatus = "Warning"
	}
	factors = append(factors, FactorScore{
		Name:    "Storage Space",
		Score:   round2(diskScore),
		Status:  diskStatus,
		Details: fmt.Sprintf("%d GB Free", input.FreeDiskGB),
	})

	// 4. Battery Factor (if applicable)
	if input.BatteryPct > 0 {
		batteryScore := 100.0 - input.BatteryDegradation
		if input.BatteryDegradation > 25.0 {
			recs = append(recs, Recommendation{
				ID:          "battery_wear",
				Severity:    "info",
				Title:       "Battery Capacity Wear",
				Description: fmt.Sprintf("Battery maximum capacity has degraded by %.1f%%.", input.BatteryDegradation),
			})
		}
		batteryScore = math.Max(0.0, math.Min(100.0, batteryScore))
		factors = append(factors, FactorScore{
			Name:    "Battery Health",
			Score:   round2(batteryScore),
			Status:  "Good",
			Details: fmt.Sprintf("%.0f%% Capacity Retained", 100.0-input.BatteryDegradation),
		})
	}

	// 5. Network Factor
	netScore := 100.0
	netStatus := "Optimal"
	if !input.NetworkConnected {
		netScore = 0.0
		netStatus = "Disconnected"
		recs = append(recs, Recommendation{
			ID:          "net_offline",
			Severity:    "critical",
			Title:       "No Network Connection",
			Description: "Your device currently has no active internet connection.",
		})
	} else if input.PingLatencyMs > 150 {
		netScore = 70.0
		netStatus = "High Latency"
	}
	factors = append(factors, FactorScore{
		Name:    "Network Health",
		Score:   round2(netScore),
		Status:  netStatus,
		Details: fmt.Sprintf("%.0f ms Latency", input.PingLatencyMs),
	})

	// Calculate overall weighted score
	totalWeight := 0.0
	sumScore := 0.0

	for _, f := range factors {
		w := 1.0
		if f.Name == "CPU Performance" || f.Name == "Memory Capacity" {
			w = 1.5
		}
		sumScore += f.Score * w
		totalWeight += w
	}

	overall := sumScore / totalWeight
	overall = round2(overall)

	statusText := "System Healthy"
	if overall < 60 {
		statusText = "Action Required"
	} else if overall < 80 {
		statusText = "Fair Condition"
	}

	return DiagnosticReport{
		OverallScore:    overall,
		StatusText:      statusText,
		Factors:         factors,
		Recommendations: recs,
	}
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}
