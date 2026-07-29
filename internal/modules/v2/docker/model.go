package docker

type ContainerInfo struct {
	ID     string   `json:"id"`
	Names  []string `json:"names"`
	Image  string   `json:"image"`
	State  string   `json:"state"`
	Status string   `json:"status"`
}

type ImageInfo struct {
	ID       string   `json:"id"`
	RepoTags []string `json:"repo_tags"`
	Size     int64    `json:"size"`
	Created  int64    `json:"created"`
}

type VolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
}

type NetworkInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
}

type SystemInfo struct {
	Containers        int    `json:"containers"`
	ContainersRunning int    `json:"containers_running"`
	ContainersStopped int    `json:"containers_stopped"`
	Images            int    `json:"images"`
	MemTotal          int64  `json:"mem_total"`
	NCPU              int    `json:"ncpu"`
	ServerVersion     string `json:"server_version"`
}

// Request payloads
type PullImageRequest struct {
	Image string `json:"image"` // ex: "nginx:latest"
}
