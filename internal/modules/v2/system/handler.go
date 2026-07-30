package system

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"serveoapi/internal/core/response"
	"serveoapi/internal/core/stream"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func getSystemStats() SystemResponse {
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

	return SystemResponse{
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
	resp := getSystemStats()
	response.SendJSON(w, http.StatusOK, resp)
}

// StreamSystem godoc
// @Summary      Stream System Metrics (SSE)
// @Description  Streams CPU, RAM, Disk, Network stats via Server-Sent Events every second
// @Tags         system
// @Produce      text/event-stream
// @Security     ApiKeyAuth
// @Success      200  {string}  string "Event Stream"
// @Router       /v2/system/stream [get]
func (hd *Handler) StreamSystem(w http.ResponseWriter, r *http.Request) {
	stream.SetupSSEHeaders(w)
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			resp := getSystemStats()
			jsonBytes, _ := json.Marshal(resp)
			stream.SendSSEEvent(w, string(jsonBytes))
			time.Sleep(1 * time.Second)
		}
	}
}
