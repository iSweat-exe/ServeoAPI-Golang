package system

import (
	"net/http"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"gorm.io/gorm"
	"serveoapi/internal/core/response"
)

type Handler struct {
	DB *gorm.DB
}

// GetSystem godoc
// @Summary      Get System Metrics
// @Description  Returns system information like CPU, RAM, Disk, Network, etc.
// @Tags         system
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  SystemResponse
// @Router       /v2/system/ [get]
func (hd *Handler) GetSystem(w http.ResponseWriter, r *http.Request) {
	// Mémoire
	v, _ := mem.VirtualMemory()

	// CPU (snapshot non bloquant)
	c, _ := cpu.Percent(0, false)
	cpuUsage := 0.0
	if len(c) > 0 {
		cpuUsage = c[0]
	}

	// Disque (volume racine généralement / sur linux, C: sur windows)
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

	// Réseau
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

	// Informations de l'hôte
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

	response.SendJSON(w, http.StatusOK, resp)
}
