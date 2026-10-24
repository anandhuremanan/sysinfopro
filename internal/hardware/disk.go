package hardware

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"time"

	"myproject/internal/storage"
	"myproject/internal/winps"
)

//go:embed disk.ps1
var diskScript string

// DriveHealth is what the drive and Windows report about a drive's condition.
// Nothing here is estimated: fields the drive does not report are left empty.
type DriveHealth struct {
	Drive string `json:"drive"` // "C:"
	Model string `json:"model"`
	Type  string `json:"type"` // NVMe SSD, SSD, HDD or ""
	// Healthy, Warning, Unhealthy or Unknown
	Status string `json:"status"`
	// Why Status is not Healthy, in plain words
	Reason string `json:"reason"`
	// From the drive's own health log (NVMe drives only); nil otherwise
	NVMe *storage.NVMeHealth `json:"nvme"`
	// 100 minus the share of rated endurance used; -1 when the drive does not say
	LifeLeftPct int `json:"life_left_pct"`
}

type diskRow struct {
	Drive  string `json:"drive"`
	Model  string `json:"model"`
	Health string `json:"health"`
}

// GetDriveHealth reports the health of every drive with a letter
func GetDriveHealth(ctx context.Context) ([]DriveHealth, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	out, err := winps.Run(ctx, diskScript)
	if err != nil {
		return nil, err
	}
	var rows []diskRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, err
	}

	results := make([]DriveHealth, 0, len(rows))
	for _, row := range rows {
		nvme, _ := storage.ReadNVMeHealth(row.Drive)
		results = append(results, judgeHealth(row, storage.DiskType(row.Drive), nvme))
	}
	return results, nil
}

// judgeHealth combines Windows' verdict with the drive's own health log; the
// worse of the two wins
func judgeHealth(row diskRow, diskType string, nvme *storage.NVMeHealth) DriveHealth {
	h := DriveHealth{
		Drive: row.Drive, Model: strings.TrimSpace(row.Model), Type: diskType,
		NVMe: nvme, LifeLeftPct: -1,
	}

	switch row.Health {
	case "Healthy", "Warning", "Unhealthy":
		h.Status = row.Health
	default:
		h.Status = "Unknown"
	}
	if h.Status == "Warning" || h.Status == "Unhealthy" {
		h.Reason = "Windows reports a problem with this drive. Back up anything important."
	}

	if nvme == nil {
		return h
	}
	h.LifeLeftPct = max(0, 100-nvme.PercentUsed)

	reason := ""
	switch {
	case nvme.CriticalWarning&0x04 != 0 || nvme.CriticalWarning&0x08 != 0:
		reason = "The drive reports it is no longer reliable or has become read-only. Back up now."
	case nvme.AvailableSpare < nvme.SpareThreshold:
		reason = "The drive has used up its spare blocks. Back up and plan to replace it."
	case nvme.PercentUsed >= 100:
		reason = "The drive has passed its rated write endurance. Keep backups current."
	case nvme.CriticalWarning&0x02 != 0:
		reason = "The drive is running too hot."
	case nvme.MediaErrors > 0:
		reason = "The drive has recorded data errors. Keep backups current."
	}
	if reason != "" {
		if h.Status == "Healthy" || h.Status == "Unknown" {
			h.Status = "Warning"
		}
		h.Reason = reason
	} else if h.Status == "Unknown" {
		// Windows did not answer but the drive's own log is clean
		h.Status = "Healthy"
	}
	return h
}
