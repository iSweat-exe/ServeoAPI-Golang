package metrics

import (
	"time"

	"serveoapi/internal/core/database"
)

// ContainerStat représente une métrique ponctuelle pour un conteneur Docker
type ContainerStat struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ContainerID   string    `gorm:"index;not null" json:"container_id"`
	Timestamp     time.Time `gorm:"index;not null" json:"timestamp"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryUsage   uint64    `json:"memory_usage"`
	MemoryLimit   uint64    `json:"memory_limit"`
	MemoryPercent float64   `json:"memory_percent"`
	NetworkRx     float64   `json:"network_rx"`
	NetworkTx     float64   `json:"network_tx"`
	BlockRead     float64   `json:"block_read"`
	BlockWrite    float64   `json:"block_write"`
}

// SystemStat représente une métrique ponctuelle pour le système hôte (Bare Metal)
type SystemStat struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Timestamp    time.Time `gorm:"index;not null" json:"timestamp"`
	CPUPercent   float64   `json:"cpu_percent"`
	MemoryUsage  uint64    `json:"memory_usage"`
	MemoryTotal  uint64    `json:"memory_total"`
	DiskUsage    uint64    `json:"disk_usage"`
	DiskTotal    uint64    `json:"disk_total"`
	NetworkRx    float64   `json:"network_rx"`
	NetworkTx    float64   `json:"network_tx"`
}

// MigrateDatabase applique les migrations pour le module de métriques
func MigrateDatabase() error {
	return database.DB.AutoMigrate(&ContainerStat{}, &SystemStat{})
}
