package files

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/docker/docker/client"

	"serveoapi/internal/core/config"
)

var (
	validServerRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	dockerClient     *client.Client
)

func init() {
	// Initialize Docker client once for performance
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err == nil {
		dockerClient = cli
	}
}

// resolveSafeRoot verifies the server exists and returns an *os.Root for atomic safe path resolution
func resolveSafeRoot(w http.ResponseWriter, r *http.Request) (*os.Root, string, bool) {
	serverName := r.PathValue("server")
	if !validServerRegex.MatchString(serverName) {
		http.Error(w, "Invalid server name format", http.StatusBadRequest)
		return nil, "", false
	}

	if dockerClient == nil {
		http.Error(w, "Docker client unavailable", http.StatusInternalServerError)
		return nil, "", false
	}

	// Validation: check if container actually exists
	_, err := dockerClient.ContainerInspect(r.Context(), serverName)
	if err != nil {
		http.Error(w, "Server not found or inaccessible", http.StatusNotFound)
		return nil, "", false
	}

	cfg := config.Load()
	rootPath := filepath.Join(cfg.AllowedMountRoot, serverName)

	// Use Go 1.24+ os.OpenRoot which guarantees atomic RESOLVE_BENEATH
	// This entirely closes the TOCTOU race window for intermediate directories
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		http.Error(w, "Security violation: Cannot open root directory safely", http.StatusForbidden)
		return nil, "", false
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = "."
	}

	// Clean the requested path to remove redundant separators
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
func ListFiles(w http.ResponseWriter, r *http.Request) {
	root, reqPath, ok := resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()

	// Safe read directly from the root
	f, err := root.Open(reqPath)
	if err != nil {
		http.Error(w, "Directory not found or cannot be read", http.StatusNotFound)
		return
	}
	defer f.Close()

	entries, err := f.Readdir(-1)
	if err != nil {
		http.Error(w, "Target is not a directory", http.StatusBadRequest)
		return
	}

	var fileList []FileInfo
	for _, info := range entries {
		// Calculate the relative path from the reqPath
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileList)
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
func ReadFile(w http.ResponseWriter, r *http.Request) {
	root, reqPath, ok := resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()

	// Atomic open safe against symlink races
	f, err := root.OpenFile(reqPath, os.O_RDONLY, 0)
	if err != nil {
		http.Error(w, "File not found or cannot be read", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
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
func WriteFile(w http.ResponseWriter, r *http.Request) {
	root, reqPath, ok := resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()

	// Atomic open safe against symlink races (O_TRUNC implies modification)
	f, err := root.OpenFile(reqPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		http.Error(w, "Cannot open file for writing: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, r.Body); err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File saved successfully"})
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
func UploadFile(w http.ResponseWriter, r *http.Request) {
	root, reqPath, ok := resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()

	// Stream multipart directly without buffering into RAM/tmpfs
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "Invalid multipart request", http.StatusBadRequest)
		return
	}

	part, err := reader.NextPart()
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer part.Close()

	cleanFileName := filepath.Base(part.FileName())
	if cleanFileName == "." || cleanFileName == "/" || cleanFileName == "" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	finalPath := filepath.Join(reqPath, cleanFileName)

	// Atomic file creation within the secure root
	f, err := root.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		http.Error(w, "Cannot create file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// True 1-pass zero-memory streaming to disk
	if _, err := io.Copy(f, part); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
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
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	root, reqPath, ok := resolveSafeRoot(w, r)
	if !ok {
		return
	}
	defer root.Close()

	if reqPath == "." || reqPath == "/" {
		http.Error(w, "Cannot delete root directory", http.StatusForbidden)
		return
	}

	// Atomic recursive removal safe against TOCTOU
	if err := root.RemoveAll(reqPath); err != nil {
		http.Error(w, "Cannot delete file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "File deleted successfully"})
}
