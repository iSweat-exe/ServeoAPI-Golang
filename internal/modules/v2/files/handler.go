package files

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/docker/docker/client"
	"gorm.io/gorm"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/response"
)

type Handler struct {
	DB        *gorm.DB
	Config    *config.Config
	DockerCli *client.Client
}

var validServerRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// resolveSafeRoot vérifie que le serveur existe et retourne un *os.Root pour une résolution de chemin atomique et sécurisée
func (h *Handler) resolveSafeRoot(
	w http.ResponseWriter,
	r *http.Request,
) (*os.Root, string, bool) {
	serverName := r.PathValue("server")
	if !validServerRegex.MatchString(serverName) {
		response.SendError(w, http.StatusBadRequest, "Invalid server name format")
		return nil, "", false
	}

	if h.DockerCli == nil {
		response.SendError(w, http.StatusInternalServerError, "Docker client unavailable")
		return nil, "", false
	}

	// Validation : vérifier si le conteneur existe vraiment
	_, err := h.DockerCli.ContainerInspect(r.Context(), serverName)
	if err != nil {
		response.SendError(w, http.StatusNotFound, "Server not found or inaccessible")
		return nil, "", false
	}

	rootPath := filepath.Join(h.Config.AllowedMountRoot, serverName)

	// Utilisation de os.OpenRoot (Go 1.24+) qui garantit RESOLVE_BENEATH de façon atomique
	// Cela ferme entièrement la fenêtre de vulnérabilité TOCTOU pour les répertoires intermédiaires
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		response.SendError(
			w,
			http.StatusForbidden,
			"Security violation: Cannot open root directory safely",
		)
		return nil, "", false
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = "."
	}

	// Nettoyer le chemin demandé pour supprimer les séparateurs redondants
	reqPath = filepath.Clean(reqPath)

	return root, reqPath, true
}

// ListFiles godoc
// @Summary      List Files
// @Description  List files and directories in a given path
// @Tags         files
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        server path string true "Server Name"
// @Param        path query string false "Relative path"
// @Success      200  {array}   FileInfo
// @Router       /v2/files/{server}/list [get]
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	root, reqPath, ok := h.resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()

	// Lecture sécurisée directement depuis la racine
	f, err := root.Open(reqPath)
	if err != nil {
		response.SendError(w, http.StatusNotFound, "Directory not found or cannot be read")
		return
	}
	defer f.Close()

	entries, err := f.Readdir(-1)
	if err != nil {
		response.SendError(w, http.StatusBadRequest, "Target is not a directory")
		return
	}

	var fileList []FileInfo
	for _, info := range entries {
		// Calculer le chemin relatif depuis reqPath
		rel := filepath.Join(reqPath, info.Name())
		if reqPath == "." || reqPath == "/" {
			rel = info.Name()
		}

		fileList = append(fileList, FileInfo{
			Name:    info.Name(),
			Size:    info.Size(),
			IsDir:   info.IsDir(),
			ModTime: info.ModTime(),
			Path:    "/" + filepath.ToSlash(rel),
		})
	}

	response.SendJSON(w, http.StatusOK, fileList)
}

func (h *Handler) withSafeRoot(w http.ResponseWriter, r *http.Request, fn func(*os.Root, string)) {
	root, reqPath, ok := h.resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()
	fn(root, reqPath)
}

// ReadFile godoc
// @Summary      Read File
// @Description  Returns the content of a file
// @Tags         files
// @Produce      octet-stream
// @Security     ApiKeyAuth
// @Param        server path string true "Server Name"
// @Param        path query string true "Relative path to file"
// @Router       /v2/files/{server}/read [get]
func (h *Handler) ReadFile(w http.ResponseWriter, r *http.Request) {
	h.withSafeRoot(w, r, func(root *os.Root, reqPath string) {
		f, err := root.OpenFile(reqPath, os.O_RDONLY, 0)
		if err != nil {
			response.SendError(w, http.StatusNotFound, "File not found or cannot be read")
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.Copy(w, f)
	})
}

// WriteFile godoc
// @Summary      Write File
// @Description  Writes raw text content to a file
// @Tags         files
// @Accept       plain
// @Produce      json
// @Security     ApiKeyAuth
// @Param        server path string true "Server Name"
// @Param        path query string true "Relative path to file"
// @Router       /v2/files/{server}/write [post]
func (h *Handler) WriteFile(w http.ResponseWriter, r *http.Request) {
	h.withSafeRoot(w, r, func(root *os.Root, reqPath string) {
		f, err := root.OpenFile(reqPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			response.SendError(
				w,
				http.StatusInternalServerError,
				"Failed to open file for writing",
			)
			return
		}
		defer f.Close()

		if _, err := io.Copy(f, r.Body); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Failed to write content")
			return
		}

		response.SendJSON(w, http.StatusOK, map[string]string{"message": "File written"})
	})
}

// UploadFile godoc
// @Summary      Upload File
// @Description  Uploads a binary file using multipart/form-data
// @Tags         files
// @Accept       mpfd
// @Produce      json
// @Security     ApiKeyAuth
// @Param        server path string true "Server Name"
// @Param        path query string true "Relative destination directory path"
// @Router       /v2/files/{server}/upload [post]
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	root, reqPath, ok := h.resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()

	// Streaming multipart direct sans mise en mémoire tampon dans la RAM ou tmpfs
	reader, err := r.MultipartReader()
	if err != nil {
		response.SendError(w, http.StatusBadRequest, "Invalid multipart request")
		return
	}

	part, err := reader.NextPart()
	if err != nil {
		response.SendError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer part.Close()

	cleanFileName := filepath.Base(part.FileName())
	if cleanFileName == "." || cleanFileName == "/" || cleanFileName == "" {
		response.SendError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	finalPath := filepath.Join(reqPath, cleanFileName)

	// Création atomique de fichier au sein de la racine sécurisée
	f, err := root.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Cannot create file: "+err.Error())
		return
	}
	defer f.Close()

	// Véritable streaming vers le disque en une seule passe et sans consommation de mémoire
	if _, err := io.Copy(f, part); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error saving file")
		return
	}

	response.SendJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// DeleteFile godoc
// @Summary      Delete File
// @Description  Deletes a file or directory
// @Tags         files
// @Produce      json
// @Security     ApiKeyAuth
// @Param        server path string true "Server Name"
// @Param        path query string true "Relative path to file or directory"
// @Router       /v2/files/{server}/delete [delete]
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	h.withSafeRoot(w, r, func(root *os.Root, reqPath string) {
		if reqPath == "." {
			response.SendError(w, http.StatusForbidden, "Cannot delete root directory")
			return
		}

		if err := root.Remove(reqPath); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Failed to delete file/folder")
			return
		}

		response.SendJSON(w, http.StatusOK, map[string]string{"message": "Deleted"})
	})
}
