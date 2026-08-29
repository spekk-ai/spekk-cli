package sandbox

import (
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

// Name reports the provider name stored in SandboxMeta.Provider.
func (p *DOProvider) Name() string { return "digitalocean" }

// Create provisions a DigitalOcean droplet with an SSH key and cloud-init,
// then records the droplet id, key id, resolved region and size, resolved
// project name, IP, and local key path on meta.
func (p *DOProvider) Create(name string, opts CreateOptions, meta *SandboxMeta) error {
	region := opts.Region
	if region == "" {
		region = "nyc1"
	}
	size := opts.Size
	if size == "" {
		size = "s-2vcpu-4gb"
	}
	// Record the resolved values, not the raw flags, so `list` and `status`
	// report what the droplet actually is when the flags were omitted.
	meta.Region = region
	meta.Size = size

	// Resolve the project before creating billable resources.
	var projectID, projectName string
	if opts.Project != "" {
		var err error
		projectID, projectName, err = resolveProject(p.client, opts.Project)
		if err != nil {
			return err
		}
		meta.Project = projectName
	}

	// Generate SSH key pair.
	fmt.Fprintln(os.Stderr, "Generating SSH key pair...")
	keyPath, err := generateSSHKeyPair(name)
	if err != nil {
		return fmt.Errorf("generating SSH key: %w", err)
	}

	pubKeyData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return fmt.Errorf("reading public key: %w", err)
	}
	pubKey := strings.TrimSpace(string(pubKeyData))

	// Upload public key to DO.
	keyName := fmt.Sprintf("spekk-%s", name)
	doKey, err := p.client.CreateSSHKey(keyName, pubKey)
	if err != nil {
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		return fmt.Errorf("uploading SSH key to DO: %w", err)
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
	if len(opts.CloudInit) > 0 {
		cloudInit = renderCloudInit(opts.CloudInit, pubKey)
	}

	// Create droplet.
	dropletName := "spekk-" + name
	fmt.Fprintf(os.Stderr, "Creating droplet %q in %s (%s)...\n", dropletName, region, size)
	droplet, err := p.client.CreateDroplet(CreateDropletRequest{
		Name:     dropletName,
		Region:   region,
		Size:     size,
		SSHKeys:  sshKeyIDs,
		VpcUUID:  opts.VPC,
		UserData: cloudInit,
	})
	if err != nil {
		if delErr := p.client.DeleteSSHKey(doKey.ID); delErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove orphaned SSH key %d from DO: %s\n", doKey.ID, delErr)
		}
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		return fmt.Errorf("creating droplet: %w", err)
	}

	// Record the identifiers as soon as they exist. A failure after this
	// point leaves a real droplet running, and the caller prints the id.
	meta.DropletID = droplet.ID
	meta.SSHKeyID = doKey.ID
	meta.SSHKeyPath = keyPath
	fmt.Fprintf(os.Stderr, "Droplet created (ID: %d). Waiting for it to become active...\n", droplet.ID)

	// Wait for the droplet to get a public IP.
	ip, err := waitForDroplet(p.client, droplet.ID)
	if err != nil {
		return err
	}
	meta.IP = ip
	fmt.Fprintf(os.Stderr, "Droplet active at %s\n", ip)

	// Assign to project.
	if projectID != "" {
		fmt.Fprintf(os.Stderr, "Assigning droplet to project %q...\n", projectName)
		urn := fmt.Sprintf("do:droplet:%d", droplet.ID)
		if err := p.client.AssignToProject(projectID, []string{urn}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not assign to project: %s\n", err)
		}
	}

	return nil
}

// Destroy deletes the droplet and its SSH key from DigitalOcean.
func (p *DOProvider) Destroy(meta *SandboxMeta) error {
	if meta.DropletID == 0 {
		return fmt.Errorf("no droplet id recorded for this sandbox; refusing to destroy, because a droplet may still be running and billing")
	}

	if err := p.client.DeleteDroplet(meta.DropletID); err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
			fmt.Fprintf(os.Stderr, "Warning: Droplet %d was already deleted.\n", meta.DropletID)
		} else {
			return fmt.Errorf("deleting droplet: %w", err)
		}
	}

	if meta.SSHKeyID != 0 {
		if err := p.client.DeleteSSHKey(meta.SSHKeyID); err != nil {
			if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
				fmt.Fprintf(os.Stderr, "Warning: SSH key %d was already deleted.\n", meta.SSHKeyID)
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
func (p *DOProvider) Status(meta *SandboxMeta) (string, error) {
	if meta.DropletID == 0 {
		return "", nil
	}
	droplet, err := p.client.GetDroplet(meta.DropletID)
	if err != nil {
		return "", err
	}
	if droplet == nil {
		return "unknown", nil
	}
	return droplet.Status, nil
}
