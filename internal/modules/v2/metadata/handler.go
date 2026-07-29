package metadata

import (
	"net/http"
	"runtime"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

// GetMetadata godoc
// @Summary      Get API Metadata
// @Description  Returns metadata about the API including author, version, etc.
// @Tags         metadata
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  MetadataResponse
// @Router       /v2/metadata/ [get]
func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	resp := MetadataResponse{
		Author:          "iSweat",
		APIVersion:      config.AppVersion,
		GoVersion:       runtime.Version(),
		Deprecated:      false,
		GithubLink:      "https://github.com/isweat-exe/ServeoAPI-Golang",
		ProtocolVersion: 2,
		CommitHash:      config.CommitHash,
		DeprecationInfo: "",
	}

	response.SendJSON(w, http.StatusOK, resp)
}
