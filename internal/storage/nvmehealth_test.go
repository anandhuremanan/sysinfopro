package storage

import (
	"encoding/binary"
	"testing"
)

func TestParseNVMeHealthLog(t *testing.T) {
	log := make([]byte, 512)
	log[0] = 0x02                                     // temperature warning
	binary.LittleEndian.PutUint16(log[1:], 327)       // 54 C in kelvin
	log[3], log[4], log[5] = 98, 10, 13               // spare, threshold, percent used
	binary.LittleEndian.PutUint64(log[48:], 92773000) // data units written
	binary.LittleEndian.PutUint64(log[128:], 4639)    // power-on hours
	binary.LittleEndian.PutUint64(log[160:], 2)       // media errors

	got := parseNVMeHealthLog(log)
	want := NVMeHealth{PercentUsed: 13, AvailableSpare: 98, SpareThreshold: 10, TemperatureC: 54,
		PowerOnHours: 4639, MediaErrors: 2, CriticalWarning: 2, TBWritten: 47.5}
	if *got != want {
		t.Errorf("got %+v\nwant %+v", *got, want)
	}
}
