// Package batteryreport reads Windows' own battery report (powercfg
// /batteryreport), which records the battery's real capacity, as the battery
// itself reports it, every week since Windows was installed.
package batteryreport

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Battery struct {
	Name      string  `json:"name"`
	Chemistry string  `json:"chemistry"`
	DesignMWh int     `json:"design_mwh"`
	FullMWh   int     `json:"full_mwh"`
	HealthPct float64 `json:"health_pct"` // full charge capacity as a share of new
	// charge cycles as the battery reports them; -1 when it does not report them
	Cycles int `json:"cycles"`
}

// Point is the capacity in one week (or day) of the history
type Point struct {
	Date      string  `json:"date"` // YYYY-MM-DD, local
	FullMWh   int     `json:"full_mwh"`
	DesignMWh int     `json:"design_mwh"`
	HealthPct float64 `json:"health_pct"`
}

// Session is one stretch on battery power
type Session struct {
	Start   string  `json:"start"` // local time, 2006-01-02T15:04:05
	Minutes int     `json:"minutes"`
	UsedPct float64 `json:"used_pct"` // share of a full charge used
	// how long a full charge would last at this rate, in minutes
	FullChargeMinutes int `json:"full_charge_minutes"`
}

type Report struct {
	HasBattery bool      `json:"has_battery"`
	Batteries  []Battery `json:"batteries"`
	HealthPct  float64   `json:"health_pct"`
	// Windows' estimate of time on battery with the screen on, now and when new
	RuntimeNowMin int       `json:"runtime_now_min"`
	RuntimeNewMin int       `json:"runtime_new_min"`
	History       []Point   `json:"history"`
	Sessions      []Session `json:"sessions"`
	TrackedSince  string    `json:"tracked_since"` // first date in the history
	// capacity lost per year, in percentage points of the original; 0 when too little history
	LossPerYear float64 `json:"loss_per_year"`
	Summary     string  `json:"summary"`
}

// the parts of the report XML that are used

type xmlReport struct {
	Batteries []struct {
		ID           string `xml:"Id"`
		Manufacturer string `xml:"Manufacturer"`
		Chemistry    string `xml:"Chemistry"`
		Design       int    `xml:"DesignCapacity"`
		Full         int    `xml:"FullChargeCapacity"`
		Cycles       int    `xml:"CycleCount"`
	} `xml:"Batteries>Battery"`
	Runtime struct {
		Design struct {
			Active string `xml:"ActiveRuntime"`
		} `xml:"DesignCapacity"`
		Full struct {
			Active string `xml:"ActiveRuntime"`
		} `xml:"FullChargeCapacity"`
	} `xml:"RuntimeEstimates"`
	History []struct {
		Start  string `xml:"LocalStartDate,attr"`
		Design int    `xml:"DesignCapacity,attr"`
		Full   int    `xml:"FullChargeCapacity,attr"`
	} `xml:"History>HistoryEntry"`
	Drains []struct {
		Start     string `xml:"LocalStartTimestamp,attr"`
		End       string `xml:"LocalEndTimestamp,attr"`
		StartMWh  int    `xml:"StartChargeCapacity,attr"`
		EndMWh    int    `xml:"EndChargeCapacity,attr"`
		StartFull int    `xml:"StartFullChargeCapacity,attr"`
	} `xml:"EnergyDrains>Drain"`
}

var isoDuration = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?)?$`)

// minutes reads an ISO 8601 duration such as "PT2H46M6S" or "P1DT3H"
func minutes(d string) int {
	m := isoDuration.FindStringSubmatch(strings.TrimSpace(d))
	if m == nil {
		return 0
	}
	n := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
	return int(math.Round(n(m[1])*24*60 + n(m[2])*60 + n(m[3]) + n(m[4])/60))
}

func pct(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return math.Round(float64(part)/float64(whole)*1000) / 10
}

const localLayout = "2006-01-02T15:04:05"

// Parse turns the XML of `powercfg /batteryreport /xml` into a Report
func Parse(r io.Reader) (*Report, error) {
	var x xmlReport
	if err := xml.NewDecoder(r).Decode(&x); err != nil {
		return nil, fmt.Errorf("could not read the battery report: %w", err)
	}
	rep := &Report{Batteries: []Battery{}, History: []Point{}, Sessions: []Session{}}

	var design, full int
	for _, b := range x.Batteries {
		if b.Design <= 0 {
			continue
		}
		name := strings.TrimSpace(strings.TrimSpace(b.Manufacturer) + " " + strings.TrimSpace(b.ID))
		cycles := b.Cycles
		if cycles <= 0 {
			cycles = -1 // many batteries never report it; 0 would be a guess
		}
		chem := map[string]string{"LION": "Lithium-ion", "LiP": "Lithium polymer", "LIP": "Lithium polymer", "PbAc": "Lead-acid", "NiMH": "Nickel-metal hydride"}[b.Chemistry]
		if chem == "" {
			chem = b.Chemistry
		}
		rep.Batteries = append(rep.Batteries, Battery{Name: name, Chemistry: chem, DesignMWh: b.Design, FullMWh: b.Full, HealthPct: pct(b.Full, b.Design), Cycles: cycles})
		design += b.Design
		full += b.Full
	}
	rep.HasBattery = len(rep.Batteries) > 0
	if !rep.HasBattery {
		rep.Summary = "This PC has no battery."
		return rep, nil
	}
	rep.HealthPct = pct(full, design)
	rep.RuntimeNowMin = minutes(x.Runtime.Full.Active)
	rep.RuntimeNewMin = minutes(x.Runtime.Design.Active)

	for _, h := range x.History {
		if h.Design <= 0 || h.Full <= 0 || len(h.Start) < 10 {
			continue
		}
		rep.History = append(rep.History, Point{Date: h.Start[:10], FullMWh: h.Full, DesignMWh: h.Design, HealthPct: pct(h.Full, h.Design)})
	}
	if len(rep.History) > 0 {
		rep.TrackedSince = rep.History[0].Date
		rep.LossPerYear = lossPerYear(rep.History)
	}

	for _, d := range x.Drains {
		start, err1 := time.Parse(localLayout, d.Start)
		end, err2 := time.Parse(localLayout, d.End)
		if err1 != nil || err2 != nil || d.StartFull <= 0 {
			continue
		}
		mins := int(end.Sub(start).Minutes())
		used := pct(d.StartMWh-d.EndMWh, d.StartFull)
		// only real use: at least 10 minutes and 2% of a charge
		if mins < 10 || used < 2 {
			continue
		}
		rep.Sessions = append(rep.Sessions, Session{Start: d.Start, Minutes: mins, UsedPct: used,
			FullChargeMinutes: int(math.Round(float64(mins) * 100 / used))})
	}
	// newest first, at most 12
	for i, j := 0, len(rep.Sessions)-1; i < j; i, j = i+1, j-1 {
		rep.Sessions[i], rep.Sessions[j] = rep.Sessions[j], rep.Sessions[i]
	}
	if len(rep.Sessions) > 12 {
		rep.Sessions = rep.Sessions[:12]
	}

	rep.Summary = summary(rep)
	return rep, nil
}

// lossPerYear fits a straight line through the history (least squares), which
// is steadier than comparing the first and last weeks, and needs 90 days or more
func lossPerYear(h []Point) float64 {
	first, err1 := time.Parse("2006-01-02", h[0].Date)
	last, err2 := time.Parse("2006-01-02", h[len(h)-1].Date)
	if err1 != nil || err2 != nil || last.Sub(first) < 90*24*time.Hour || len(h) < 4 {
		return 0
	}
	var sx, sy, sxx, sxy float64
	n := 0.0
	for _, p := range h {
		t, err := time.Parse("2006-01-02", p.Date)
		if err != nil {
			continue
		}
		x := t.Sub(first).Hours() / 24 / 365.25
		sx += x
		sy += p.HealthPct
		sxx += x * x
		sxy += x * p.HealthPct
		n++
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return 0
	}
	slope := (n*sxy - sx*sy) / den
	if slope >= 0 {
		return 0 // capacity readings can drift up after calibration
	}
	return math.Round(-slope*10) / 10
}

func summary(r *Report) string {
	h := r.HealthPct
	var s string
	switch {
	case h >= 90:
		s = fmt.Sprintf("The battery holds %.0f%% of its original charge: like new.", h)
	case h >= 80:
		s = fmt.Sprintf("The battery holds %.0f%% of its original charge: normal wear.", h)
	case h >= 65:
		s = fmt.Sprintf("The battery holds %.0f%% of its original charge: worn, but still useful.", h)
	default:
		s = fmt.Sprintf("The battery holds only %.0f%% of its original charge: consider replacing it.", h)
	}
	if r.LossPerYear > 0 {
		s += fmt.Sprintf(" It is losing about %.0f%% a year.", math.Max(1, r.LossPerYear))
	}
	return s
}
