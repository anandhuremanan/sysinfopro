package functions

import (
	"testing"

	"github.com/distatus/battery"
)

func TestBatteryDetailsMapping(t *testing.T) {
	b := &battery.Battery{State: battery.State{Raw: battery.Discharging}, Current: 20000, Full: 40000, Design: 50000, ChargeRate: 10000}
	d := detailsFrom(b)
	if d.Remaining != 2 {
		t.Errorf("remaining = %v hours, want 2 (20000 mWh at 10000 mW)", d.Remaining)
	}
	if d.CurrentCapacity != 40000 || d.ChargeNow != 20000 || d.Percentage != 50 {
		t.Errorf("capacity %v, charge %v, percent %v", d.CurrentCapacity, d.ChargeNow, d.Percentage)
	}
	if d.DegradationPct != 20 || d.HealthStatus != "Fair" {
		t.Errorf("wear %v, health %q", d.DegradationPct, d.HealthStatus)
	}

	charging := detailsFrom(&battery.Battery{State: battery.State{Raw: battery.Charging}, Current: 20000, Full: 40000, ChargeRate: 10000})
	if charging.Remaining != 0 {
		t.Errorf("no time left is shown while charging, got %v", charging.Remaining)
	}
	unknown := detailsFrom(&battery.Battery{State: battery.State{Raw: battery.Discharging}, Current: 20000})
	if unknown.HealthStatus != "" || unknown.Percentage != 0 {
		t.Errorf("unknown capacities must stay unknown: %+v", unknown)
	}
}
