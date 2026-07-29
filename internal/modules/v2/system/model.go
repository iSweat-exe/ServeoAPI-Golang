package system

type SystemResponse struct {
	CPUUsage  float64 `json:"cpu_usage"`
	RAMTotal  uint64  `json:"ram_total"`
	RAMUsed   uint64  `json:"ram_used"`
	DiskTotal uint64  `json:"disk_total"`
	DiskUsed  uint64  `json:"disk_used"`
	NetworkTX uint64  `json:"network_tx"`
	NetworkRX uint64  `json:"network_rx"`
	Ping          string  `json:"ping"`
	Arch          string  `json:"arch"`
	OS            string  `json:"os"`
	Platform      string  `json:"platform"`
	KernelVersion string  `json:"kernel_version"`
	Hostname      string  `json:"hostname"`
	Uptime        uint64  `json:"uptime"`
}
