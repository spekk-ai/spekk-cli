package sandbox

// CreateResult holds the output of a successful Provider.Create call.
type CreateResult struct {
	IP         string // public IP of the created VM
	InstanceID string // opaque identifier — only the provider interprets it
	SSHKeyPath string // path to private key for SSH access
	Provider   string // provider name (e.g., "digitalocean", "manual")
}

// Provider abstracts VM lifecycle from sandbox orchestration.
// Implementations handle provider-specific resource creation, teardown, and
// status checks while the generic layer handles provisioning, credential
// injection, and agent deployment.
type Provider interface {
	// Create provisions a VM (or registers an existing one) and returns
	// connection details. The config map carries provider-specific settings
	// (e.g., region, size, cloud_init template) — each provider interprets
	// only the keys it cares about.
	Create(name string, config map[string]string) (*CreateResult, error)

	// Destroy tears down all provider-managed resources identified by
	// instanceID (VM, SSH keys, etc.). The instanceID is opaque — only
	// the provider that created it can interpret it.
	Destroy(instanceID string) error

	// Status returns the live VM state from the provider (e.g., "active",
	// "off", "unknown"). The instanceID is opaque.
	Status(instanceID string) (string, error)
}
