package system

import (
	"encoding/json"
	"net/http"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// GetSystem godoc
// @Summary      Get System Metrics
// @Description  Returns system information like CPU, RAM, Disk, Network, etc.
// @Tags         system
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  SystemResponse
// @Router       /v2/system/ [get]
func GetSystem(w http.ResponseWriter, r *http.Request) {
	// Mem
	v, _ := mem.VirtualMemory()

	// CPU (non-blocking snapshot)
	c, _ := cpu.Percent(0, false)
	cpuUsage := 0.0
	if len(c) > 0 {
		cpuUsage = c[0]
	}

	// Disk (root volume usually / on linux, C: on windows)
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:\\"
	}
	d, _ := disk.Usage(path)
	var diskTotal, diskUsed uint64
	if d != nil {
		diskTotal = d.Total
		diskUsed = d.Used
	}

	// Network
	n, _ := net.IOCounters(false)
	var tx, rx uint64
	if len(n) > 0 {
		tx = n[0].BytesSent
		rx = n[0].BytesRecv
	}

	var ramTotal, ramUsed uint64
	if v != nil {
		ramTotal = v.Total
		ramUsed = v.Used
	}

	// Host Info
	h, _ := host.Info()
	var platform, kernel, hostname string
	var uptime uint64
	if h != nil {
		platform = h.Platform + " " + h.PlatformVersion
		kernel = h.KernelVersion
		hostname = h.Hostname
		uptime = h.Uptime
	}

	resp := SystemResponse{
		CPUUsage:      cpuUsage,
		RAMTotal:      ramTotal,
		RAMUsed:       ramUsed,
		DiskTotal:     diskTotal,
		DiskUsed:      diskUsed,
		NetworkTX:     tx,
		NetworkRX:     rx,
		Ping:          "pong",
		Arch:          runtime.GOARCH,
		OS:            runtime.GOOS,
		Platform:      platform,
		KernelVersion: kernel,
		Hostname:      hostname,
		Uptime:        uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
