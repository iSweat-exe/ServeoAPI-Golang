package metadata

import (
	"encoding/json"
	"net/http"
	"runtime"
)

// GetMetadata godoc
// @Summary      Get API Metadata
// @Description  Returns metadata about the API including author, version, etc.
// @Tags         metadata
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  MetadataResponse
// @Router       /v2/metadata/ [get]
func GetMetadata(w http.ResponseWriter, r *http.Request) {
	resp := MetadataResponse{
		Author:          "iSweat",
		APIVersion:      "2.0.0",
		GoVersion:       runtime.Version(),
		Deprecated:      false,
		GithubLink:      "https://github.com/isweat-exe/ServeoAPI-Golang",
		ProtocolVersion: 2,
		CommitHash:      "unknown", // En production, ceci pourrait être injecté via des ldflags au build
		DeprecationInfo: "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
