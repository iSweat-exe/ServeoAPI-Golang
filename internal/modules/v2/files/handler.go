package files

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

	// Le répertoire de données peut ne pas encore exister (conteneur jamais démarré,
	// bind mount pas encore matérialisé par Docker, serveur tout juste créé, etc.).
	// Le conteneur étant déjà vérifié ci-dessus, on le crée à la volée plutôt que
	// d'échouer avec une erreur qui ressemble à un problème de sécurité.
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		response.SendError(
			w,
			http.StatusInternalServerError,
			"Cannot create data directory for this server: "+err.Error(),
		)
		return nil, "", false
	}

	// Utilisation de os.OpenRoot (Go 1.24+) qui garantit RESOLVE_BENEATH de façon atomique
	// Cela ferme entièrement la fenêtre de vulnérabilité TOCTOU pour les répertoires intermédiaires
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		response.SendError(
			w,
			http.StatusInternalServerError,
			"Cannot access data directory for this server: "+err.Error(),
		)
		return nil, "", false
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = "."
	}

	// Nettoyer le chemin demandé
	reqPath = filepath.Clean(reqPath)

	// os.Root exige un chemin strictement relatif (sans slash au début)
	// et utilise "." pour désigner la racine elle-même.
	reqPath = filepath.ToSlash(reqPath)
	for len(reqPath) > 0 && reqPath[0] == '/' {
		reqPath = reqPath[1:]
	}
	if reqPath == "" {
		reqPath = "."
	}

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

		// Appliquer les permissions de la racine au fichier
		chownFileToMatchRoot(f, root)

		if _, err := io.Copy(f, r.Body); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Failed to write content")
			return
		}

		response.SendJSON(w, http.StatusOK, map[string]string{"message": "File written"})
	})
}

// UploadFile godoc
// @Summary      Upload File
// @Description  Uploads a binary file using multipart/form-data. The multipart
// @Description  filename may include a relative subpath (e.g. "mods/foo.jar")
// @Description  to preserve folder structure from drag-and-drop uploads;
// @Description  intermediate directories are created automatically.
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

	relPath, ok := sanitizeUploadPath(part.FileName())
	if !ok {
		response.SendError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	finalPath := filepath.Join(reqPath, relPath)

	// Le nom de fichier multipart peut contenir un sous-chemin (upload de dossier
	// par drag-and-drop) : on crée les répertoires intermédiaires si nécessaire.
	if dir := filepath.Dir(finalPath); dir != "." {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			response.SendError(
				w,
				http.StatusInternalServerError,
				"Cannot create directory: "+err.Error(),
			)
			return
		}
	}

	// Création atomique de fichier au sein de la racine sécurisée
	f, err := root.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		response.SendError(w, http.StatusInternalServerError, "Cannot create file: "+err.Error())
		return
	}
	defer f.Close()

	// Appliquer les permissions de la racine au fichier uploadé
	chownFileToMatchRoot(f, root)

	// Véritable streaming vers le disque en une seule passe et sans consommation de mémoire
	if _, err := io.Copy(f, part); err != nil {
		response.SendError(w, http.StatusInternalServerError, "Error saving file")
		return
	}

	response.SendJSON(w, http.StatusOK, map[string]string{"message": "File uploaded successfully"})
}

// sanitizeUploadPath nettoie le nom de fichier fourni par le client multipart
// (potentiellement un sous-chemin relatif, ex: "mods/foo.jar", pour un upload
// de dossier par drag-and-drop) et rejette toute tentative de traversée.
// os.Root refuse déjà toute évasion de la racine ; cette validation ne sert
// qu'à renvoyer une erreur explicite plutôt que de dépendre uniquement de lui.
func sanitizeUploadPath(name string) (string, bool) {
	cleaned := filepath.ToSlash(filepath.Clean(name))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	if base := filepath.Base(cleaned); base == "" || base == "." || base == "/" {
		return "", false
	}
	return cleaned, true
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

		if err := root.RemoveAll(reqPath); err != nil {
			response.SendError(w, http.StatusInternalServerError, "Failed to delete file/folder")
			return
		}

		response.SendJSON(w, http.StatusOK, map[string]string{"message": "Deleted"})
	})
}
