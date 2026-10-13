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

	// Where the owning process came from; empty when Windows does not allow reading it
	CommandLine string `json:"command_line"`
	WorkingDir  string `json:"working_dir"`
	ProjectName string `json:"project_name"`
	ProjectRoot string `json:"project_root"`
	GitBranch   string `json:"git_branch"`
	StartedAt   int64  `json:"started_at"` // unix seconds, 0 if unknown
}

// processOrigin is what we can learn about a process beyond its name
type processOrigin struct {
	name        string
	commandLine string
	workingDir  string
	project     *Project
	startedAt   int64
}

func describeProcess(pid int32) processOrigin {
	p, err := process.NewProcess(pid)
	if err != nil {
		return processOrigin{name: "System/Unknown"}
	}

	origin := processOrigin{}
	origin.name, _ = p.Name()
	if origin.name == "" {
		// protected system processes do not give up their name
		origin.name = "System/Unknown"
	}
	origin.commandLine, _ = p.Cmdline()
	origin.workingDir, _ = p.Cwd()
	if created, err := p.CreateTime(); err == nil {
		origin.startedAt = created / 1000
	}
	origin.project = DetectProject(origin.workingDir)
	return origin
}

func GetActiveSockets() ([]SocketItem, error) {
	conns, err := net.Connections("all")
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate network connections: %w", err)
	}

	// Cache process details by PID
	procCache := make(map[int32]processOrigin)

	results := []SocketItem{}
	for _, c := range conns {
		if c.Status != "LISTEN" && c.Status != "ESTABLISHED" {
			continue
		}

		origin, ok := procCache[c.Pid]
		if !ok {
			origin = describeProcess(c.Pid)
			procCache[c.Pid] = origin
		}

		protocol := "TCP"
		if c.Type == 2 {
			protocol = "UDP"
		}

		item := SocketItem{
			PID:         c.Pid,
			ProcessName: origin.name,
			Protocol:    protocol,
			LocalIP:     c.Laddr.IP,
			LocalPort:   c.Laddr.Port,
			RemoteIP:    c.Raddr.IP,
			RemotePort:  c.Raddr.Port,
			Status:      c.Status,
			CommandLine: origin.commandLine,
			WorkingDir:  origin.workingDir,
			StartedAt:   origin.startedAt,
		}
		if origin.project != nil {
			item.ProjectName = origin.project.Name
			item.ProjectRoot = origin.project.Root
			item.GitBranch = origin.project.Branch
		}
		results = append(results, item)
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
