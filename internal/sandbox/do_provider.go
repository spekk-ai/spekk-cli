package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// DOProvider implements Provider using the DigitalOcean API.
type DOProvider struct {
	client *Client
}

// NewDOProvider creates a DOProvider backed by a DO API client.
// The client reads its token from DO_API_TOKEN or DIGITALOCEAN_TOKEN.
func NewDOProvider() (*DOProvider, error) {
	client, err := NewClient()
	if err != nil {
		return nil, err
	}
	return &DOProvider{client: client}, nil
}

// doInstanceState is the provider-specific state encoded in the opaque InstanceID.
type doInstanceState struct {
	DropletID int `json:"droplet_id"`
	SSHKeyID  int `json:"ssh_key_id"`
}

func encodeInstanceState(s doInstanceState) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decodeInstanceState(instanceID string) (doInstanceState, error) {
	var s doInstanceState
	if err := json.Unmarshal([]byte(instanceID), &s); err != nil {
		return s, fmt.Errorf("decoding DO instance state: %w", err)
	}
	return s, nil
}

// Create provisions a DigitalOcean droplet with an SSH key and cloud-init.
// Config keys: "region", "size", "vpc", "project", "cloud_init".
func (p *DOProvider) Create(name string, config map[string]string) (*CreateResult, error) {
	region := config["region"]
	if region == "" {
		region = "nyc1"
	}
	size := config["size"]
	if size == "" {
		size = "s-2vcpu-4gb"
	}

	// Resolve project before creating billable resources.
	var projectID, projectName string
	if config["project"] != "" {
		var err error
		projectID, projectName, err = resolveProject(p.client, config["project"])
		if err != nil {
			return nil, err
		}
	}

	// Generate SSH key pair.
	fmt.Fprintln(os.Stderr, "Generating SSH key pair...")
	keyPath, err := generateSSHKeyPair(name)
	if err != nil {
		return nil, fmt.Errorf("generating SSH key: %w", err)
	}

	pubKeyData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return nil, fmt.Errorf("reading public key: %w", err)
	}
	pubKey := strings.TrimSpace(string(pubKeyData))

	// Upload public key to DO.
	keyName := fmt.Sprintf("spekk-%s", name)
	doKey, err := p.client.CreateSSHKey(keyName, pubKey)
	if err != nil {
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		return nil, fmt.Errorf("uploading SSH key to DO: %w", err)
	}
	fmt.Fprintf(os.Stderr, "SSH key uploaded to DigitalOcean (ID: %d)\n", doKey.ID)

	// Collect all account SSH keys so the user can also SSH in.
	existingKeys, listErr := p.client.ListSSHKeys()
	if listErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not list existing SSH keys (%s); only the generated key will be authorized\n", listErr)
	}
	sshKeyIDs := []int{doKey.ID}
	for _, k := range existingKeys {
		if k.ID != doKey.ID {
			sshKeyIDs = append(sshKeyIDs, k.ID)
		}
	}

	// Render cloud-init with the generated public key.
	cloudInit := ""
	if tpl := config["cloud_init"]; tpl != "" {
		cloudInit = renderCloudInit([]byte(tpl), pubKey)
	}

	// Create droplet.
	dropletName := "spekk-" + name
	fmt.Fprintf(os.Stderr, "Creating droplet %q in %s (%s)...\n", dropletName, region, size)
	droplet, err := p.client.CreateDroplet(CreateDropletRequest{
		Name:     dropletName,
		Region:   region,
		Size:     size,
		SSHKeys:  sshKeyIDs,
		VpcUUID:  config["vpc"],
		UserData: cloudInit,
	})
	if err != nil {
		if delErr := p.client.DeleteSSHKey(doKey.ID); delErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove orphaned SSH key %d from DO: %s\n", doKey.ID, delErr)
		}
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		return nil, fmt.Errorf("creating droplet: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Droplet created (ID: %d). Waiting for it to become active...\n", droplet.ID)

	// Wait for the droplet to get a public IP.
	ip, err := waitForDroplet(p.client, droplet.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\nDroplet ID: %d -- not auto-destroyed, debug manually.\n", err, droplet.ID)
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "Droplet active at %s\n", ip)

	// Assign to project.
	if projectID != "" {
		fmt.Fprintf(os.Stderr, "Assigning droplet to project %q...\n", projectName)
		urn := fmt.Sprintf("do:droplet:%d", droplet.ID)
		if err := p.client.AssignToProject(projectID, []string{urn}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not assign to project: %s\n", err)
		}
	}

	return &CreateResult{
		IP:         ip,
		InstanceID: encodeInstanceState(doInstanceState{DropletID: droplet.ID, SSHKeyID: doKey.ID}),
		SSHKeyPath: keyPath,
		Provider:   "digitalocean",
	}, nil
}

// Destroy deletes the droplet and its SSH key from DigitalOcean.
func (p *DOProvider) Destroy(instanceID string) error {
	state, err := decodeInstanceState(instanceID)
	if err != nil {
		return err
	}

	if err := p.client.DeleteDroplet(state.DropletID); err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
			fmt.Fprintf(os.Stderr, "Warning: Droplet %d was already deleted.\n", state.DropletID)
		} else {
			return fmt.Errorf("deleting droplet: %w", err)
		}
	}

	if state.SSHKeyID != 0 {
		if err := p.client.DeleteSSHKey(state.SSHKeyID); err != nil {
			if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
				fmt.Fprintf(os.Stderr, "Warning: SSH key %d was already deleted.\n", state.SSHKeyID)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: could not remove SSH key from DO: %s\n", err)
			}
		} else {
			fmt.Fprintln(os.Stderr, "SSH key removed from DigitalOcean.")
		}
	}

	return nil
}

// Status fetches the live droplet status from the DO API.
func (p *DOProvider) Status(instanceID string) (string, error) {
	state, err := decodeInstanceState(instanceID)
	if err != nil {
		return "", err
	}

	droplet, err := p.client.GetDroplet(state.DropletID)
	if err != nil {
		return "", err
	}
	if droplet == nil {
		return "unknown", nil
	}
	return droplet.Status, nil
}
