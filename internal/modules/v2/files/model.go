package files

import "time"

// FileInfo represents metadata for a file or directory
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
	ModTime time.Time `json:"mod_time"`
	Path    string    `json:"path"` // relative path to the root
}
