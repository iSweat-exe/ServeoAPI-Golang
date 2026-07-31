// Package permissions centralizes the catalog of RBAC scopes exposed by the
// API modules, so callers (e.g. the frontend permissions picker) don't have
// to hardcode scope strings that only otherwise live in each module's routes.go.
package permissions

// Scope is a single grantable permission string, e.g. "docker.containers.read".
type Scope struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Group bundles the scopes that apply to a single resource.
type Group struct {
	Resource string  `json:"resource"`
	Label    string  `json:"label"`
	Scopes   []Scope `json:"scopes"`
}

// Catalog lists every permission scope registered across the API modules,
// grouped by resource. Keep this in sync with the RequirePermission(...)
// calls in each module's routes.go.
var Catalog = []Group{
	{Resource: "users", Label: "Users", Scopes: []Scope{
		{Key: "users.manage", Label: "Manage users"},
	}},
	{Resource: "docker.containers", Label: "Containers", Scopes: []Scope{
		{Key: "docker.containers.read", Label: "View"},
		{Key: "docker.containers.write", Label: "Create & control"},
		{Key: "docker.containers.delete", Label: "Delete"},
	}},
	{Resource: "docker.images", Label: "Images", Scopes: []Scope{
		{Key: "docker.images.read", Label: "View"},
		{Key: "docker.images.write", Label: "Pull"},
		{Key: "docker.images.delete", Label: "Delete"},
	}},
	{Resource: "docker.volumes", Label: "Volumes", Scopes: []Scope{
		{Key: "docker.volumes.read", Label: "View"},
		{Key: "docker.volumes.write", Label: "Create"},
		{Key: "docker.volumes.delete", Label: "Delete"},
	}},
	{Resource: "docker.networks", Label: "Networks", Scopes: []Scope{
		{Key: "docker.networks.read", Label: "View"},
		{Key: "docker.networks.write", Label: "Create"},
		{Key: "docker.networks.delete", Label: "Delete"},
	}},
	{Resource: "docker.system", Label: "Docker system", Scopes: []Scope{
		{Key: "docker.system.read", Label: "View info & events"},
		{Key: "docker.system.delete", Label: "Prune"},
	}},
	{Resource: "docker.compose", Label: "Compose", Scopes: []Scope{
		{Key: "docker.compose.write", Label: "Deploy stacks"},
	}},
	{Resource: "templates", Label: "Templates", Scopes: []Scope{
		{Key: "templates.read", Label: "View & deploy"},
	}},
	{Resource: "files", Label: "Files", Scopes: []Scope{
		{Key: "files.read", Label: "View & download"},
		{Key: "files.write", Label: "Edit, upload & delete"},
	}},
	{Resource: "backups", Label: "Backups", Scopes: []Scope{
		{Key: "backups.read", Label: "View"},
		{Key: "backups.write", Label: "Create & restore"},
	}},
	{Resource: "metrics", Label: "Metrics", Scopes: []Scope{
		{Key: "metrics.read", Label: "View history"},
	}},
	{Resource: "system", Label: "System", Scopes: []Scope{
		{Key: "system.read", Label: "View host stats"},
	}},
	{Resource: "metadata", Label: "Metadata", Scopes: []Scope{
		{Key: "metadata.read", Label: "View"},
	}},
	{Resource: "ovh", Label: "OVH", Scopes: []Scope{
		{Key: "ovh.read", Label: "View servers"},
		{Key: "ovh.write", Label: "Reboot servers"},
	}},
	{Resource: "mcp", Label: "MCP", Scopes: []Scope{
		{Key: "mcp.use", Label: "Use MCP server"},
	}},
}
