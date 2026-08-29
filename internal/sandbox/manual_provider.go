package sandbox

import "fmt"

// ManualProvider implements Provider for pre-existing SSH-reachable machines.
type ManualProvider struct{}

func (p *ManualProvider) Create(name string, config map[string]string) (*CreateResult, error) {
	ip := config["ip"]
	if ip == "" {
		return nil, fmt.Errorf("manual provider requires --ip flag")
	}
	sshKey := config["ssh_key"]
	if sshKey == "" {
		return nil, fmt.Errorf("manual provider requires --ssh-key flag")
	}

	return &CreateResult{
		IP:         ip,
		InstanceID: "", // manual provider has no cloud instance
		SSHKeyPath: sshKey,
		Provider:   "manual",
	}, nil
}

func (p *ManualProvider) Destroy(instanceID string) error {
	// Manual provider has no cloud resources to tear down.
	// Agent service stop is handled by the generic Destroy flow.
	return nil
}

func (p *ManualProvider) Status(instanceID string) (string, error) {
	// Manual provider has no API to query — status comes from SSH checks only.
	return "manual", nil
}
