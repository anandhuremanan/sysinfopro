package functions

type SystemInfo struct {
	CPUModel        string  `json:"cpu_model"`
	Manufacturer    string  `json:"cpu_vendor"`
	Cores           int     `json:"cpu_cores"`
	CPUFreqGHz      float64 `json:"cpu_freq_ghz"`
	LoggedInUser    string  `json:"logged_in_user"`
	TotalRAMMB      uint64  `json:"total_ram_mb"`
	FreeRAMMB       uint64  `json:"free_ram_mb"`
	DiskType        string  `json:"disk_type"`
	TotalDiskGB     uint64  `json:"total_disk_gb"`
	FreeDiskGB      uint64  `json:"free_disk_gb"`
	DiskUsagePct    float64 `json:"disk_usage_percent"`
	NetworkStatus   string  `json:"network_status"`
	OS              string  `json:"host_os"`
	Platform        string  `json:"host_platform"`
	PlatformVersion string  `json:"host_platform_version"`
}

type InternetSpeedStat struct {
	DownloadSpeed float64 `json:"download_speed"`
	UploadSpeed   float64 `json:"upload_speed"`
	PingLatencyMs float64 `json:"ping_latency_ms"`
	JitterMs      float64 `json:"jitter_ms"`
	PacketLossPct float64 `json:"packet_loss_pct"`
}

type MemoryStats struct {
	TotalRAMMB    uint64  `json:"totalRAMMB"`
	FreeRAMMB     uint64  `json:"freeRAMMB"`
	UsedRAMMB     uint64  `json:"usedRAMMB"`
	UsagePercent  float64 `json:"usagePercent"`
	Timestamp     int64   `json:"timestamp"`
	SwapTotalMB   uint64  `json:"swapTotalMB"`
	SwapUsedMB    uint64  `json:"swapUsedMB"`
	PressureLevel string  `json:"pressureLevel"`
}

type CPUStats struct {
	Usage         float64 `json:"usage"`
	Timestamp     int64   `json:"timestamp"`
	FrequencyGHz  float64 `json:"frequency_ghz"`
	IsThrottling  bool    `json:"is_throttling"`
	SustainedLoad bool    `json:"sustained_load"`
}
