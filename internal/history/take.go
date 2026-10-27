package history

import (
	"fmt"
	"time"

	"sysinfopro/internal/network"
	"sysinfopro/internal/system"
)

// Take records startup programs, installed software, listening ports and PATH
// as they are now. A list that cannot be read is left nil (unknown) rather than
// empty, so What changed never reports everything in it as removed.
func Take() Snapshot {
	snapshot := Snapshot{
		TakenAt: time.Now(),
		Apps:    system.InstalledApps(),
	}
	if items, err := system.GetStartupItems(); err == nil {
		snapshot.Startup = []string{}
		for _, item := range items {
			snapshot.Startup = append(snapshot.Startup, fmt.Sprintf("%s (%s)", item.Name, item.Location))
		}
	}
	if ports, err := network.ListeningPorts(); err == nil {
		snapshot.Ports = ports
	}
	if analysis, err := system.GetEnvPathAnalysis(); err == nil && analysis != nil {
		snapshot.Path = []string{}
		for _, entry := range analysis.PathEntries {
			snapshot.Path = append(snapshot.Path, entry.Path)
		}
	}
	return snapshot
}
