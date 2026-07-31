package metrics

import (
	"context"
	"encoding/json"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"serveoapi/internal/core/database"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

const (
	// retentionWindow est la durée de conservation des métriques historisées.
	retentionWindow = 24 * time.Hour
	// maxConcurrentStats borne les appels simultanés à l'API Docker durant une collecte.
	maxConcurrentStats = 8
)

func StartMetricsWorker(ctx context.Context, interval time.Duration, dockerCli *client.Client) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Metrics worker stopping...")
				return
			case <-ticker.C:
				collectSystemMetrics()
				collectContainerMetrics(ctx, dockerCli)
			}
		}
	}()
}

func collectSystemMetrics() {
	v, err := mem.VirtualMemory()
	if err != nil {
		slog.Warn("Lecture de la mémoire impossible", "error", err)
	}

	c, err := cpu.Percent(0, false)
	if err != nil {
		slog.Warn("Lecture du CPU impossible", "error", err)
	}
	cpuUsage := 0.0
	if len(c) > 0 {
		cpuUsage = c[0]
	}

	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:\\"
	}
	d, err := disk.Usage(path)
	if err != nil {
		slog.Warn("Lecture du disque impossible", "error", err, "path", path)
	}

	n, err := net.IOCounters(false)
	if err != nil {
		slog.Warn("Lecture du réseau impossible", "error", err)
	}
	var tx, rx float64
	if len(n) > 0 {
		tx = float64(n[0].BytesSent)
		rx = float64(n[0].BytesRecv)
	}

	stat := SystemStat{
		Timestamp:   time.Now(),
		CPUPercent:  cpuUsage,
		MemoryUsage: 0,
		MemoryTotal: 0,
		DiskUsage:   0,
		DiskTotal:   0,
		NetworkRx:   rx,
		NetworkTx:   tx,
	}

	if v != nil {
		stat.MemoryUsage = v.Used
		stat.MemoryTotal = v.Total
	}
	if d != nil {
		stat.DiskUsage = d.Used
		stat.DiskTotal = d.Total
	}

	if err := database.DB.Create(&stat).Error; err != nil {
		slog.Error("Failed to save system metrics", "error", err)
	}

	purgeOldRecords(&SystemStat{})
}

// purgeOldRecords supprime les enregistrements de plus de 24h du modèle fourni.
func purgeOldRecords(model interface{}) {
	if err := database.DB.
		Where("timestamp < ?", time.Now().Add(-retentionWindow)).
		Delete(model).Error; err != nil {
		slog.Error("Purge des anciennes métriques impossible", "error", err)
	}
}

type ContainerStatsData struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage  uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64            `json:"usage"`
		Limit uint64            `json:"limit"`
		Stats map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Major uint64 `json:"major"`
			Minor uint64 `json:"minor"`
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
}

func collectContainerMetrics(ctx context.Context, dockerCli *client.Client) {
	if dockerCli == nil {
		return
	}

	containers, err := dockerCli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		slog.Error("Failed to list containers for metrics", "error", err)
		return
	}

	// Chaque appel de stats attend Docker : on les parallélise avec une concurrence bornée
	// pour que la collecte ne dépasse pas l'intervalle du worker.
	sem := make(chan struct{}, maxConcurrentStats)
	var wg sync.WaitGroup

	for _, c := range containers {
		wg.Add(1)
		go func(c container.Summary) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			processContainerStats(ctx, dockerCli, c)
		}(c)
	}
	wg.Wait()

	purgeOldRecords(&ContainerStat{})
}

func processContainerStats(ctx context.Context, cli *client.Client, c container.Summary) {
	stats, err := cli.ContainerStats(ctx, c.ID, false)
	if err != nil {
		return
	}
	defer stats.Body.Close()

	var v ContainerStatsData
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return
	}

	cpuDelta := float64(
		v.CPUStats.CPUUsage.TotalUsage,
	) - float64(
		v.PreCPUStats.CPUUsage.TotalUsage,
	)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(
			len(v.CPUStats.CPUUsage.PercpuUsage),
		) * 100.0
	}

	memUsage := float64(v.MemoryStats.Usage) - float64(v.MemoryStats.Stats["cache"])
	memLimit := float64(v.MemoryStats.Limit)
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = (memUsage / memLimit) * 100.0
	}

	var rx, tx float64
	for _, network := range v.Networks {
		rx += float64(network.RxBytes)
		tx += float64(network.TxBytes)
	}

	var blkRead, blkWrite float64
	for _, io := range v.BlkioStats.IoServiceBytesRecursive {
		if len(io.Op) > 0 && (io.Op[0] == 'r' || io.Op[0] == 'R') {
			blkRead += float64(io.Value)
		} else if len(io.Op) > 0 && (io.Op[0] == 'w' || io.Op[0] == 'W') {
			blkWrite += float64(io.Value)
		}
	}

	id := c.ID
	if len(id) > 12 {
		id = id[:12]
	}

	stat := ContainerStat{
		ContainerID:   id,
		Timestamp:     time.Now(),
		CPUPercent:    cpuPercent,
		MemoryUsage:   uint64(memUsage),
		MemoryLimit:   uint64(memLimit),
		MemoryPercent: memPercent,
		NetworkRx:     rx,
		NetworkTx:     tx,
		BlockRead:     blkRead,
		BlockWrite:    blkWrite,
	}

	if err := database.DB.Create(&stat).Error; err != nil {
		slog.Error("Failed to save container metrics", "error", err)
	}
}
