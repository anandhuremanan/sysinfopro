// Package vdisk finds the virtual disks of WSL distributions and Docker Desktop,
// which grow as they are used and never shrink on their own, and reports how much
// space Docker could give back.
package vdisk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Disk is one virtual disk file
type Disk struct {
	Name    string `json:"name"`    // distribution name, or "Docker Desktop"
	Kind    string `json:"kind"`    // wsl | docker
	Path    string `json:"path"`    // the .vhdx file
	Default bool   `json:"default"` // the default WSL distribution
	Version int    `json:"version"` // WSL 1 or 2; 0 for Docker's own disk
	// SizeBytes is the file's size; OnDiskBytes is the space it really takes,
	// which is less for a sparse disk
	SizeBytes   int64 `json:"size_bytes"`
	OnDiskBytes int64 `json:"on_disk_bytes"`
	Sparse      bool  `json:"sparse"`
	// registered with WSL, but the disk file is gone
	Missing bool `json:"missing"`
}

// distro is one WSL registration, as stored in the registry
type distro struct {
	name, basePath, vhdName string
	version                 int
	isDefault               bool
}

func isDocker(name string) bool { return strings.HasPrefix(strings.ToLower(name), "docker-desktop") }

// ListDisks returns every WSL distribution's disk and Docker Desktop's disks, largest first
func ListDisks() []Disk {
	disks := []Disk{}
	seen := map[string]bool{}

	for _, d := range registeredDistros() {
		kind := "wsl"
		if isDocker(d.name) {
			kind = "docker"
		}
		disk := Disk{Name: d.name, Kind: kind, Default: d.isDefault, Version: d.version}
		if d.version == 1 {
			// WSL 1 keeps files in a plain folder, not a virtual disk
			disk.Path = filepath.Join(d.basePath, "rootfs")
			disk.Missing = !exists(disk.Path)
			disks = append(disks, disk)
			continue
		}
		disk.Path = filepath.Join(d.basePath, d.vhdName)
		measure(&disk)
		seen[strings.ToLower(disk.Path)] = true
		disks = append(disks, disk)
	}

	// newer Docker Desktop keeps its data disk outside any registered distribution
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		for _, rel := range []string{`Docker\wsl\disk\docker_data.vhdx`, `Docker\wsl\data\ext4.vhdx`, `Docker\wsl\main\ext4.vhdx`} {
			path := filepath.Join(local, rel)
			if seen[strings.ToLower(path)] || !exists(path) {
				continue
			}
			disk := Disk{Name: "Docker Desktop", Kind: "docker", Path: path}
			measure(&disk)
			disks = append(disks, disk)
		}
	}

	sort.SliceStable(disks, func(i, j int) bool { return disks[i].OnDiskBytes > disks[j].OnDiskBytes })
	return disks
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func measure(disk *Disk) {
	info, err := os.Stat(disk.Path)
	if err != nil {
		disk.Missing = true
		return
	}
	disk.SizeBytes = info.Size()
	disk.OnDiskBytes, disk.Sparse = allocated(disk.Path, info.Size())
}

// --- Docker ---

type DockerRow struct {
	Type        string `json:"type"` // Images, Containers, Local Volumes, Build Cache
	Count       string `json:"count"`
	Size        string `json:"size"`
	Reclaimable string `json:"reclaimable"`
}

type DockerUsage struct {
	Installed bool        `json:"installed"` // the docker command was found
	Running   bool        `json:"running"`   // and Docker answered
	Rows      []DockerRow `json:"rows"`
}

func dockerCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "docker", args...)
	hideWindow(cmd)
	return cmd
}

// GetDockerUsage asks Docker how much space images, containers, volumes and
// build cache use, and how much of it is unused
func GetDockerUsage(ctx context.Context) DockerUsage {
	usage := DockerUsage{Rows: []DockerRow{}}
	if _, err := exec.LookPath("docker"); err != nil {
		return usage
	}
	usage.Installed = true

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := dockerCommand(ctx, "system", "df", "--format", "{{json .}}").Output()
	if err != nil {
		return usage // installed, but Docker is not running
	}
	usage.Running = true
	usage.Rows = parseDockerDF(out)
	return usage
}

func parseDockerDF(out []byte) []DockerRow {
	rows := []DockerRow{}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		var line struct {
			Type, TotalCount, Size, Reclaimable string
		}
		if json.Unmarshal(scanner.Bytes(), &line) == nil && line.Type != "" {
			rows = append(rows, DockerRow{line.Type, line.TotalCount, line.Size, line.Reclaimable})
		}
	}
	return rows
}

var reclaimedPattern = regexp.MustCompile(`(?i)total reclaimed space:\s*(\S+)`)

// PruneDocker removes stopped containers, unused networks, dangling images and
// build cache (docker system prune). Volumes, which can hold data, are kept.
// It returns the space Docker reports as reclaimed, such as "1.2GB".
func PruneDocker(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	out, err := dockerCommand(ctx, "system", "prune", "--force").CombinedOutput()
	if err != nil {
		return "", &commandError{strings.TrimSpace(string(out)), err}
	}
	if m := reclaimedPattern.FindSubmatch(out); m != nil {
		return string(m[1]), nil
	}
	return "0B", nil
}

type commandError struct {
	output string
	err    error
}

func (e *commandError) Error() string {
	if e.output != "" {
		return e.output
	}
	return e.err.Error()
}
