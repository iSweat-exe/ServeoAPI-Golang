package metadata

type MetadataResponse struct {
	Author          string `json:"author"`
	APIVersion      string `json:"api_version"`
	GoVersion       string `json:"go_version"`
	Deprecated      bool   `json:"deprecated"`
	GithubLink      string `json:"github_link"`
	ProtocolVersion int    `json:"protocol_version"`
	CommitHash      string `json:"commit_hash"`
	DeprecationInfo string `json:"deprecation_info"`
}
