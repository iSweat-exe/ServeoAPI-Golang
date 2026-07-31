package system

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"serveoapi/internal/core/permissions"
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

// logProbeError trace une sonde système indisponible sans interrompre la réponse :
// les champs concernés restent à zéro.
func logProbeError(probe string, err error) {
	if err != nil {
		slog.Warn("Sonde système indisponible", "probe", probe, "error", err)
	}
}

func getSystemStats() SystemResponse {
	// Mémoire
	v, err := mem.VirtualMemory()
	logProbeError("memory", err)

	// CPU (snapshot non bloquant)
	c, err := cpu.Percent(0, false)
	logProbeError("cpu", err)
	cpuUsage := 0.0
	if len(c) > 0 {
		cpuUsage = c[0]
	}

	// Disque (volume racine généralement / sur linux, C: sur windows)
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:\\"
	}
	d, err := disk.Usage(path)
	logProbeError("disk", err)
	var diskTotal, diskUsed uint64
	if d != nil {
		diskTotal = d.Total
		diskUsed = d.Used
	}

	// Réseau
	n, err := net.IOCounters(false)
	logProbeError("network", err)
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
	h, err := host.Info()
	logProbeError("host", err)
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
			jsonBytes, err := json.Marshal(resp)
			if err != nil {
				slog.Error("Sérialisation des métriques système impossible", "error", err)
				return
			}
			stream.SendSSEEvent(w, string(jsonBytes))
			time.Sleep(1 * time.Second)
		}
	}
}

// GetPermissionsCatalog godoc
// @Summary      List available permission scopes
// @Description  Returns the catalog of grantable permission scopes, grouped by resource, used to build a permissions picker (requires users.manage)
// @Tags         system
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}  permissions.Group
// @Router       /v2/system/permissions [get]
func (hd *Handler) GetPermissionsCatalog(w http.ResponseWriter, r *http.Request) {
	response.SendJSON(w, http.StatusOK, permissions.Catalog)
}
