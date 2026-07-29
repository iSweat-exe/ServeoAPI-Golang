package metrics

import (
	"context"
	"encoding/json"
	"log"
	"runtime"
	"time"

	"serveoapi/internal/core/database"
	"github.com/docker/docker/client"

	"github.com/docker/docker/api/types/container"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func StartMetricsWorker(ctx context.Context, interval time.Duration, dockerCli *client.Client) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Metrics worker stopping...")
				return
			case <-ticker.C:
				collectSystemMetrics()
				collectContainerMetrics(dockerCli)
			}
		}
	}()
}

func collectSystemMetrics() {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Percent(0, false)
	cpuUsage := 0.0
	if len(c) > 0 {
		cpuUsage = c[0]
	}

	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:\\"
	}
	d, _ := disk.Usage(path)
	
	n, _ := net.IOCounters(false)
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
		log.Printf("Failed to save system metrics: %v", err)
	}

	// Nettoyage des anciens enregistrements (plus de 24h)
	database.DB.Where("timestamp < ?", time.Now().Add(-24*time.Hour)).Delete(&SystemStat{})
}

func collectContainerMetrics(dockerCli *client.Client) {
	cli := dockerCli
	if cli == nil {
		return
	}
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		log.Printf("Failed to list containers for metrics: %v", err)
		return
	}

	for _, c := range containers {
		stats, err := cli.ContainerStats(ctx, c.ID, false)
		if err != nil {
			continue
		}
		
		var v struct {
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
		
		if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
			stats.Body.Close()
			continue
		}
		stats.Body.Close()

		cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
		cpuPercent := 0.0
		if systemDelta > 0.0 && cpuDelta > 0.0 {
			cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
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
			log.Printf("Failed to save container metrics: %v", err)
		}
	}

	// Nettoyage des anciens enregistrements (plus de 24h)
	database.DB.Where("timestamp < ?", time.Now().Add(-24*time.Hour)).Delete(&ContainerStat{})
}
