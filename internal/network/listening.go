package network

import (
	"fmt"
	"sort"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// ListeningPorts returns every listening TCP port as "port process", sorted and de-duplicated.
// Unlike GetActiveSockets it is not capped, so it suits before/after comparisons.
func ListeningPorts() ([]string, error) {
	conns, err := net.Connections("tcp")
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate network connections: %w", err)
	}

	names := map[int32]string{}
	seen := map[string]bool{}
	type entry struct {
		port  uint32
		label string
	}
	entries := []entry{}

	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		name, ok := names[c.Pid]
		if !ok {
			name = "unknown"
			if p, err := process.NewProcess(c.Pid); err == nil {
				if n, err := p.Name(); err == nil && n != "" {
					name = n
				}
			}
			names[c.Pid] = name
		}
		label := fmt.Sprintf("%d %s", c.Laddr.Port, name)
		if !seen[label] {
			seen[label] = true
			entries = append(entries, entry{c.Laddr.Port, label})
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].port < entries[j].port })
	labels := make([]string, len(entries))
	for i, e := range entries {
		labels[i] = e.label
	}
	return labels, nil
}
