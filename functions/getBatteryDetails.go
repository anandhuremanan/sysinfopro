package functions

import (
	"fmt"
	"math"

	"github.com/distatus/battery"
)

type BatteryDetails struct {
	Percentage float64 `json:"percentage"`
	State      string  `json:"state"`
	// hours left at the current drain; 0 when not discharging or unknown
	Remaining      float64 `json:"remaining"`
	DesignCapacity float64 `json:"design_capacity"` // mWh, when new
	// how much the battery holds when fully charged now (mWh)
	CurrentCapacity float64 `json:"current_capacity"`
	// how much charge is in it right now (mWh)
	ChargeNow      float64 `json:"charge_now"`
	DegradationPct float64 `json:"degradation_pct"`
	// Always -1: Windows' battery API gives no cycle count here, and the old
	// estimate from wear was a guess. The Battery tool reads the real count when
	// the battery reports one.
	CycleEstimate int `json:"cycle_estimate"`
	// "" when the battery does not report its design or full-charge capacity
	HealthStatus string `json:"health_status"`
}

func GetBatteryDetails() (*BatteryDetails, error) {
	batteries, err := battery.GetAll()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve battery info: %w", err)
	}
	if len(batteries) == 0 {
		return nil, fmt.Errorf("no battery detected")
	}
	return detailsFrom(batteries[0]), nil
}

// detailsFrom turns the library's reading into what the screens show
func detailsFrom(b *battery.Battery) *BatteryDetails {
	percentage := 0.0
	if b.Full > 0 {
		percentage = (b.Current / b.Full) * 100
	}

	remaining := 0.0
	if b.State.Raw == battery.Discharging && b.ChargeRate > 0 {
		remaining = b.Current / b.ChargeRate
	}

	// wear: full-charge capacity against the capacity it was designed for
	degradation := 0.0
	healthStatus := ""
	if b.Design > 0 && b.Full > 0 {
		if b.Full < b.Design {
			degradation = ((b.Design - b.Full) / b.Design) * 100.0
		}
		healthStatus = "Excellent"
		if degradation > 30.0 {
			healthStatus = "Service Recommended"
		} else if degradation > 15.0 {
			healthStatus = "Fair"
		}
	}

	return &BatteryDetails{
		Percentage:      percentage,
		State:           b.State.String(),
		Remaining:       remaining,
		DesignCapacity:  b.Design,
		CurrentCapacity: b.Full,
		ChargeNow:       b.Current,
		DegradationPct:  math.Round(degradation*10) / 10,
		CycleEstimate:   -1,
		HealthStatus:    healthStatus,
	}
}
