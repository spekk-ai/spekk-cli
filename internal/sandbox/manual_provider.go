package sandbox

import "fmt"

// ManualProvider implements Provider for pre-existing SSH-reachable machines.
// The full implementation is tracked by the manual-provider assertion.
type ManualProvider struct{}

func (p *ManualProvider) Create(name string, config map[string]string) (*CreateResult, error) {
	ip := config["ip"]
	if ip == "" {
		return nil, fmt.Errorf("manual provider requires --ip flag")
	}
	sshKey := config["ssh_key"]

	return &CreateResult{
		IP:         ip,
		InstanceID: "", // manual provider has no cloud instance
		SSHKeyPath: sshKey,
		Provider:   "manual",
	}, nil
}

func (p *ManualProvider) Destroy(instanceID string) error {
	// Manual provider has no cloud resources to tear down.
	return nil
}

func (p *ManualProvider) Status(instanceID string) (string, error) {
	// Manual provider has no API to query — status comes from SSH checks only.
	return "manual", nil
}
