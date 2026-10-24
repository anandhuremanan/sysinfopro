package storage

import (
	"encoding/binary"
	"math"
)

// NVMeHealth is the drive's own SMART / health log
type NVMeHealth struct {
	PercentUsed     int     `json:"percent_used"` // share of rated endurance used; can pass 100
	AvailableSpare  int     `json:"available_spare"`
	SpareThreshold  int     `json:"spare_threshold"`
	TemperatureC    int     `json:"temperature_c"`
	PowerOnHours    int64   `json:"power_on_hours"`
	MediaErrors     int64   `json:"media_errors"`
	CriticalWarning int     `json:"critical_warning"` // bit field; 0 means none
	TBWritten       float64 `json:"tb_written"`
}

// parseNVMeHealthLog reads the fields of the 512-byte NVMe "SMART / Health
// Information" log page (NVMe base specification, log identifier 02h)
func parseNVMeHealthLog(log []byte) *NVMeHealth {
	// 128-bit counters; the low 64 bits are plenty for any real drive
	low64 := func(offset int) int64 {
		v := binary.LittleEndian.Uint64(log[offset:])
		if v > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(v)
	}

	kelvin := int(binary.LittleEndian.Uint16(log[1:]))
	temperature := 0
	if kelvin > 0 {
		temperature = kelvin - 273
	}

	// a data unit is 1000 blocks of 512 bytes
	unitsWritten := low64(48)
	return &NVMeHealth{
		CriticalWarning: int(log[0]),
		TemperatureC:    temperature,
		AvailableSpare:  int(log[3]),
		SpareThreshold:  int(log[4]),
		PercentUsed:     int(log[5]),
		TBWritten:       math.Round(float64(unitsWritten)*512000/1e12*10) / 10,
		PowerOnHours:    low64(128),
		MediaErrors:     low64(160),
	}
}
