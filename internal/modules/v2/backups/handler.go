package backups

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/response"
)

type Handler struct {
	Config *config.Config
}

type BackupInfo struct {
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) getBackupsDir() string {
	// ex. /var/serveoapi/data/ -> /var/serveoapi/backups/
	// On remonte d'un niveau par rapport à AllowedMountRoot et on ajoute backups
	parent := filepath.Dir(filepath.Clean(h.Config.AllowedMountRoot))
	return filepath.Join(parent, "backups")
}

// CreateBackup godoc
// @Summary      Create Server Backup
// @Description  Creates a zip archive of the server's data folder
// @Tags         backups
// @Produce      json
// @Security     ApiKeyAuth
// @Param        server   path      string  true  "Server name (folder name)"
// @Success      201      {object}  BackupInfo
// @Failure      404,500  {string}  string
// @Router       /v2/backups/{server} [post]
func (h *Handler) CreateBackup(
	w http.ResponseWriter,
	r *http.Request,
) {
	server := r.PathValue("server")
	if strings.Contains(server, "/") || strings.Contains(server, "\\") || server == ".." {
		response.SendError(w, http.StatusBadRequest, "Invalid server name")
		return
	}

	srcDir := filepath.Join(h.Config.AllowedMountRoot, server)
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		response.SendError(w, http.StatusNotFound, "Server directory not found")
		return
	}

	backupsDir := h.getBackupsDir()
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Failed to create backups directory")
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := server + "_" + timestamp + ".zip"
	destPath := filepath.Join(backupsDir, filename)

	if err := zipDir(srcDir, destPath); err != nil {
		response.SendError(
			w,
			http.StatusInternalServerError,
			"Failed to create backup archive: "+err.Error(),
		)
		return
	}

	info, err := os.Stat(destPath)
	if err != nil {
		response.SendError(
			w,
			http.StatusInternalServerError,
			"Failed to stat new backup: "+err.Error(),
		)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(BackupInfo{
		Filename:  info.Name(),
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
	})
}

func (h *Handler) getServerNameOrError(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	server := r.PathValue("server")
	if strings.Contains(server, "/") || strings.Contains(server, "\\") || server == ".." {
		response.SendError(w, http.StatusBadRequest, "Invalid server name")
		return "", false
	}
	return server, true
}

// ListBackups godoc
// @Summary      List Server Backups
// @Description  Lists available backups for a server
// @Tags         backups
// @Produce      json
// @Security     ApiKeyAuth
// @Param        server   path      string  true  "Server name (folder name)"
// @Success      200      {array}   BackupInfo
// @Router       /v2/backups/{server} [get]
func (h *Handler) ListBackups(
	w http.ResponseWriter,
	r *http.Request,
) {
	server, ok := h.getServerNameOrError(w, r)
	if !ok {
		return
	}

	backupsDir := h.getBackupsDir()
	entries, err := os.ReadDir(backupsDir)
	if err != nil && !os.IsNotExist(err) {
		response.SendError(w, http.StatusInternalServerError, "Failed to read backups directory")
		return
	}

	var backups []BackupInfo
	prefix := server + "_"
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) &&
			strings.HasSuffix(entry.Name(), ".zip") {
			info, err := entry.Info()
			if err == nil {
				backups = append(backups, BackupInfo{
					Filename:  info.Name(),
					Size:      info.Size(),
					CreatedAt: info.ModTime(),
				})
			}
		}
	}

	if backups == nil {
		backups = []BackupInfo{}
	}

	response.SendJSON(w, http.StatusOK, backups)
}

type RestoreRequest struct {
	Filename string `json:"filename"`
}

// RestoreBackup godoc
// @Summary      Restore Server Backup
// @Description  Restores a server backup, overwriting existing files
// @Tags         backups
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        server   path      string  true  "Server name (folder name)"
// @Param        body     body      RestoreRequest true "Restore options"
// @Success      204
// @Failure      400,404,500 {string} string
// @Router       /v2/backups/{server}/restore [post]
func (h *Handler) RestoreBackup(
	w http.ResponseWriter,
	r *http.Request,
) {
	server, ok := h.getServerNameOrError(w, r)
	if !ok {
		return
	}

	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid body")
		return
	}

	if strings.Contains(req.Filename, "/") || strings.Contains(req.Filename, "\\") ||
		req.Filename == ".." {
		response.SendError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	zipPath := filepath.Join(h.getBackupsDir(), req.Filename)
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		response.SendError(w, http.StatusNotFound, "Backup file not found")
		return
	}

	destDir := filepath.Join(h.Config.AllowedMountRoot, server)
	// Créer s'il n'existe pas
	os.MkdirAll(destDir, 0o755)

	if err := unzip(zipPath, destDir); err != nil {
		response.SendError(
			w,
			http.StatusInternalServerError,
			"Failed to restore backup: "+err.Error(),
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helpers

func zipDir(src string, dest string) error {
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Obtenir le chemin relatif
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		f, err := w.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		return err
	}
	return filepath.Walk(src, walker)
}

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Résolution sécurisée du chemin
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue // Protection contre la vulnérabilité Zip Slip
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
