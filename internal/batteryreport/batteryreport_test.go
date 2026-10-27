package batteryreport

import (
	"strings"
	"testing"
)

const sample = `<?xml version="1.0" encoding="utf-8"?>
<BatteryReport xmlns="http://schemas.microsoft.com/battery/2012">
  <Batteries>
    <Battery>
      <Id>G71C000NB110</Id><Manufacturer>GFL</Manufacturer><Chemistry>LION</Chemistry>
      <DesignCapacity>45623</DesignCapacity><FullChargeCapacity>35355</FullChargeCapacity><CycleCount>0</CycleCount>
    </Battery>
  </Batteries>
  <RuntimeEstimates>
    <DesignCapacity><Capacity>45623</Capacity><ActiveRuntime>PT3H34M20S</ActiveRuntime></DesignCapacity>
    <FullChargeCapacity><Capacity>35355</Capacity><ActiveRuntime>PT2H46M6S</ActiveRuntime></FullChargeCapacity>
  </RuntimeEstimates>
  <History>
    <HistoryEntry LocalStartDate="2024-05-13T08:52:20" DesignCapacity="45623" FullChargeCapacity="45000" />
    <HistoryEntry LocalStartDate="2024-11-13T00:00:00" DesignCapacity="45623" FullChargeCapacity="42000" />
    <HistoryEntry LocalStartDate="2025-05-13T00:00:00" DesignCapacity="45623" FullChargeCapacity="40000" />
    <HistoryEntry LocalStartDate="2025-11-13T00:00:00" DesignCapacity="45623" FullChargeCapacity="37500" />
    <HistoryEntry LocalStartDate="2026-05-13T00:00:00" DesignCapacity="45623" FullChargeCapacity="35355" />
    <HistoryEntry LocalStartDate="2026-06-01T00:00:00" DesignCapacity="0" FullChargeCapacity="0" />
  </History>
  <EnergyDrains>
    <Drain LocalStartTimestamp="2026-10-24T13:48:57" LocalEndTimestamp="2026-10-24T14:14:30"
      StartChargeCapacity="34407" StartFullChargeCapacity="35181" EndChargeCapacity="28367" EndFullChargeCapacity="35181" />
    <Drain LocalStartTimestamp="2026-10-24T15:00:00" LocalEndTimestamp="2026-10-24T15:03:00"
      StartChargeCapacity="28000" StartFullChargeCapacity="35181" EndChargeCapacity="27900" EndFullChargeCapacity="35181" />
  </EnergyDrains>
</BatteryReport>`

func TestParse(t *testing.T) {
	r, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if !r.HasBattery || len(r.Batteries) != 1 {
		t.Fatalf("batteries = %+v", r.Batteries)
	}
	b := r.Batteries[0]
	if b.HealthPct != 77.5 || b.Cycles != -1 || b.Chemistry != "Lithium-ion" || b.Name != "GFL G71C000NB110" {
		t.Errorf("battery = %+v", b)
	}
	if r.RuntimeNowMin != 166 || r.RuntimeNewMin != 214 {
		t.Errorf("runtime = %d now, %d new", r.RuntimeNowMin, r.RuntimeNewMin)
	}
	if len(r.History) != 5 || r.TrackedSince != "2024-05-13" {
		t.Errorf("history = %d points since %s (empty entries skipped)", len(r.History), r.TrackedSince)
	}
	// about 21 points lost over two years
	if r.LossPerYear < 9 || r.LossPerYear > 12 {
		t.Errorf("loss per year = %v", r.LossPerYear)
	}
	// the 3-minute blip is not a session
	if len(r.Sessions) != 1 || r.Sessions[0].Minutes != 25 || r.Sessions[0].UsedPct != 17.2 {
		t.Fatalf("sessions = %+v", r.Sessions)
	}
	if fc := r.Sessions[0].FullChargeMinutes; fc < 140 || fc > 150 {
		t.Errorf("full charge at that rate = %d min", fc)
	}
	if !strings.Contains(r.Summary, "78%") || !strings.Contains(r.Summary, "worn") || !strings.Contains(r.Summary, "a year") {
		t.Errorf("summary = %q", r.Summary)
	}
}

func TestNoBattery(t *testing.T) {
	r, err := Parse(strings.NewReader(`<BatteryReport xmlns="http://schemas.microsoft.com/battery/2012"><Batteries/></BatteryReport>`))
	if err != nil || r.HasBattery || r.Summary != "This PC has no battery." {
		t.Errorf("got %+v, %v", r, err)
	}
}

func TestMinutes(t *testing.T) {
	cases := map[string]int{"PT2H46M6S": 166, "P1DT3H44M36S": 1665, "PT45S": 1, "P867DT19H40M37S": 1249661, "junk": 0}
	for in, want := range cases {
		if got := minutes(in); got != want {
			t.Errorf("minutes(%q) = %d, want %d", in, got, want)
		}
	}
}
