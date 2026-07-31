package network

import (
	"fmt"
	"sort"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type SocketItem struct {
	PID         int32  `json:"pid"`
	ProcessName string `json:"process_name"`
	Protocol    string `json:"protocol"`
	LocalPort   uint32 `json:"local_port"`
	LocalIP     string `json:"local_ip"`
	RemoteIP    string `json:"remote_ip"`
	RemotePort  uint32 `json:"remote_port"`
	Status      string `json:"status"`
}

func GetActiveSockets() ([]SocketItem, error) {
	conns, err := net.Connections("all")
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate network connections: %w", err)
	}

	// Cache process names by PID
	procCache := make(map[int32]string)

	results := []SocketItem{}
	for _, c := range conns {
		if c.Status != "LISTEN" && c.Status != "ESTABLISHED" {
			continue
		}

		pName, ok := procCache[c.Pid]
		if !ok {
			p, err := process.NewProcess(c.Pid)
			if err == nil {
				name, _ := p.Name()
				procCache[c.Pid] = name
				pName = name
			} else {
				pName = "System/Unknown"
				procCache[c.Pid] = pName
			}
		}

		protocol := "TCP"
		if c.Type == 2 {
			protocol = "UDP"
		}

		results = append(results, SocketItem{
			PID:         c.Pid,
			ProcessName: pName,
			Protocol:    protocol,
			LocalIP:     c.Laddr.IP,
			LocalPort:   c.Laddr.Port,
			RemoteIP:    c.Raddr.IP,
			RemotePort:  c.Raddr.Port,
			Status:      c.Status,
		})
	}

	// Sort by LocalPort ascending
	sort.Slice(results, func(i, j int) bool {
		return results[i].LocalPort < results[j].LocalPort
	})

	if len(results) > 40 {
		results = results[:40]
	}

	return results, nil
}
