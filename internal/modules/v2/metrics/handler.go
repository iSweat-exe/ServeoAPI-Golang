package metrics

import (
	"net/http"
	"time"

	"serveoapi/internal/core/database"
	"serveoapi/internal/core/response"
)

type Handler struct{}

// GetContainerMetricsHistory godoc
// @Summary      Get Container Metrics History
// @Description  Returns the 24h metrics history for a specific container
// @Tags         metrics
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Container ID"
// @Success      200  {array}   ContainerStat
// @Router       /v2/metrics/history/containers/{id} [get]
func (h *Handler) GetContainerMetricsHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) > 12 {
		id = id[:12]
	}

	var stats []ContainerStat
	// Il y a 24 heures
	since := time.Now().Add(-24 * time.Hour)

	if err := database.DB.Where("container_id = ? AND timestamp >= ?", id, since).Order("timestamp asc").Find(&stats).Error; err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to retrieve metrics history")
		return
	}

	response.SendJSON(w, http.StatusOK, stats)
}

// GetSystemMetricsHistory godoc
// @Summary      Get System Metrics History
// @Description  Returns the 24h metrics history for the bare-metal system
// @Tags         metrics
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {array}   SystemStat
// @Router       /v2/metrics/history/system [get]
func (h *Handler) GetSystemMetricsHistory(w http.ResponseWriter, r *http.Request) {
	var stats []SystemStat
	since := time.Now().Add(-24 * time.Hour)

	if err := database.DB.Where("timestamp >= ?", since).Order("timestamp asc").Find(&stats).Error; err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to retrieve system metrics history")
		return
	}

	response.SendJSON(w, http.StatusOK, stats)
}
