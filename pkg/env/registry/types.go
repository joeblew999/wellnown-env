// Package registry provides types for service registration in the wellknown-env mesh.
//
// Services register themselves to a NATS KV bucket with:
// - GitHub identity (org/repo/commit/tag/branch)
// - Instance info (id, host, started time)
// - Config fields (extracted from struct with conf tags)
//
// This information enables:
// - Service discovery across the mesh
// - Config/env requirement visibility
// - Dependency tracking via service: tags
// - Change detection in CI/CD
package registry

import "time"

// ServiceRegistration is the complete registration payload sent to NATS KV.
// Key format: {org}.{repo}.{instance_id}
type ServiceRegistration struct {
	GitHub   GitHubInfo   `json:"github"`
	Instance InstanceInfo `json:"instance"`
	Fields   []FieldInfo  `json:"fields"`
}

// GitHubInfo identifies the service by its GitHub coordinates.
// Populated via ldflags at build time.
type GitHubInfo struct {
	Org    string `json:"org"`              // GitHub organization
	Repo   string `json:"repo"`             // Repository name
	Commit string `json:"commit,omitempty"` // Git commit hash
	Tag    string `json:"tag,omitempty"`    // Git tag/version
	Branch string `json:"branch,omitempty"` // Git branch
}

// InstanceInfo identifies a specific running instance of a service
type InstanceInfo struct {
	ID      string    `json:"id"`      // Unique instance ID (UUID)
	Host    string    `json:"host"`    // Host:port the service is listening on
	Started time.Time `json:"started"` // When the instance started
}

// FieldInfo describes a config field extracted from the struct via reflection
type FieldInfo struct {
	Path     string `json:"path"`               // Field path (e.g., "DB.Password")
	Type     string `json:"type"`               // Go type (string, int, bool, etc.)
	EnvKey   string `json:"env_key"`            // Environment variable name
	Default  string `json:"default,omitempty"`  // Default value if any
	Required bool   `json:"required,omitempty"` // Is field required?
	IsSecret bool   `json:"is_secret,omitempty"` // Is field a secret (masked)?

	// For service dependencies
	Dependency string `json:"dependency,omitempty"` // org/repo if this is a service: tag
}

// Build-time variables set via ldflags
// Example: go build -ldflags "-X github.com/joeblew999/wellnown-env/pkg/env/registry.GitOrg=myorg"
var (
	GitOrg    string // -X ...registry.GitOrg=joeblew999
	GitRepo   string // -X ...registry.GitRepo=my-service
	GitCommit string // -X ...registry.GitCommit=$(git rev-parse HEAD)
	GitTag    string // -X ...registry.GitTag=$(git describe --tags)
	GitBranch string // -X ...registry.GitBranch=$(git rev-parse --abbrev-ref HEAD)
)

// GetGitHubInfo returns GitHubInfo populated from ldflags
func GetGitHubInfo() GitHubInfo {
	return GitHubInfo{
		Org:    GitOrg,
		Repo:   GitRepo,
		Commit: GitCommit,
		Tag:    GitTag,
		Branch: GitBranch,
	}
}

// Name returns the service name in org/repo format
func (g GitHubInfo) Name() string {
	if g.Org == "" || g.Repo == "" {
		return ""
	}
	return g.Org + "/" + g.Repo
}

// KVKey returns the NATS KV key for this registration
func (r ServiceRegistration) KVKey() string {
	return r.GitHub.Org + "." + r.GitHub.Repo + "." + r.Instance.ID
}
