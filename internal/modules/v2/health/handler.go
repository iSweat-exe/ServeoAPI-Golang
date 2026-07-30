package health

import (
	"context"
	"net/http"
	"time"

	"serveoapi/internal/core/response"

	"github.com/docker/docker/client"
	"gorm.io/gorm"
)

type Handler struct {
	DB        *gorm.DB
	DockerCli *client.Client
}

// Check godoc
// @Summary      Health Check
// @Description  Verifies the health of the API (Database and Docker Socket)
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /health [get]
func (h *Handler) Check(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check Database
	sqlDB, err := h.DB.DB()
	if err != nil {
		response.SendError(w, http.StatusServiceUnavailable, "Database connection failed")
		return
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		response.SendError(w, http.StatusServiceUnavailable, "Database ping failed")
		return
	}

	// Check Docker
	if _, err := h.DockerCli.Ping(ctx); err != nil {
		response.SendError(w, http.StatusServiceUnavailable, "Docker daemon is unreachable")
		return
	}

	response.SendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
