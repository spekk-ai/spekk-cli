package sandbox

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// DOProvider can label its machines and make images of them.
var (
	_ Labeler = (*DOProvider)(nil)
	_ Imager  = (*DOProvider)(nil)
)

// doDefaultRegion is the region of a droplet when --region does not say.
const doDefaultRegion = "nyc1"

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
		region = doDefaultRegion
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

	// Render cloud-init with the generated public key.
	cloudInit := ""
	if len(opts.CloudInit) > 0 {
		cloudInit = renderCloudInit(opts.CloudInit, pubKey)
	}

	// Create droplet.
	dropletName := "spekk-" + name
	fmt.Fprintf(os.Stderr, "Creating droplet %q in %s (%s)...\n", dropletName, region, size)
	var image any
	if opts.Image != nil {
		id, err := strconv.Atoi(opts.Image.ImageID)
		if err != nil {
			return fmt.Errorf("image %s has ID %q, which is not a DigitalOcean image ID", opts.Image.Name, opts.Image.ImageID)
		}
		image = id
	}
	droplet, err := p.client.CreateDroplet(CreateDropletRequest{
		Name:   dropletName,
		Region: region,
		Size:   size,
		Image:  image,
		// Only the generated key. A key from the account would give its
		// holder root on the droplet, and root reads the credentials
		// spekk injects. The operator logs in with spekk sandbox ssh.
		SSHKeys:  []int{doKey.ID},
		VpcUUID:  opts.VPC,
		UserData: cloudInit,
		// The caller's labels go on at create, not after the active wait:
		// a caller that finds its droplets by a label also finds one that
		// a killed or failed create leaves.
		Tags: append([]string{"spekk-sandbox"}, sortedLabels(opts.Labels, ":")...),
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
		return fmt.Errorf("%w: a droplet may still be running and billing", ErrNoMachineRecorded)
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

// Label puts each label on the droplet as a DigitalOcean tag key:value.
func (p *DOProvider) Label(meta *SandboxMeta, labels map[string]string) error {
	if meta.DropletID == 0 {
		return fmt.Errorf("no droplet recorded")
	}
	var errs []error
	for _, tag := range sortedLabels(labels, ":") {
		if err := p.client.TagDroplet(tag, meta.DropletID); err != nil {
			errs = append(errs, fmt.Errorf("tag %s: %w", tag, err))
		}
	}
	return errors.Join(errs...)
}

// ListLabeled lists the droplets with the spekk-sandbox tag, which every
// droplet spekk creates carries. A droplet without a spekk-kind tag was
// created before labels existed, and reads as persistent.
func (p *DOProvider) ListLabeled() ([]RemoteSandbox, error) {
	droplets, err := p.client.ListDroplets("spekk-sandbox")
	if err != nil {
		return nil, err
	}
	var found []RemoteSandbox
	for _, d := range droplets {
		labels := map[string]string{}
		for _, tag := range d.Tags {
			if k, v, ok := strings.Cut(tag, ":"); ok {
				labels[k] = v
			}
		}
		meta := SandboxMeta{Provider: p.Name(), DropletID: d.ID, IP: d.PublicIP(), Region: d.Region.Slug, CreatedAt: d.CreatedAt}
		metaFromLabels(&meta, labels)
		found = append(found, RemoteSandbox{Name: strings.TrimPrefix(d.Name, "spekk-"), Meta: meta})
	}
	return found, nil
}

// ImageRegion is the region of the droplet, which must have a copy of the
// image: a DigitalOcean snapshot belongs to the regions it was copied to.
func (p *DOProvider) ImageRegion(opts CreateOptions) string {
	if opts.Region != "" {
		return opts.Region
	}
	return doDefaultRegion
}

// doActionPoll is how often the provider reads the state of an action, and
// doActionTimeout is how long it waits for one. Variables so a test can make
// them short.
var (
	doActionPoll    = 10 * time.Second
	doActionTimeout = time.Hour
)

// waitForAction waits until an action is completed, and fails when it errored
// or did not end in time.
func (p *DOProvider) waitForAction(a *Action, what string) error {
	start := time.Now()
	lastReport := start
	for {
		if a.Status == "completed" {
			return nil
		}
		if a.Status == "errored" {
			return fmt.Errorf("%s: action %d errored", what, a.ID)
		}
		if time.Since(start) > doActionTimeout {
			return fmt.Errorf("%s: action %d did not complete within %s", what, a.ID, doActionTimeout)
		}
		if time.Since(lastReport) >= time.Minute {
			lastReport = time.Now()
			fmt.Fprintf(os.Stderr, "Still waiting for %s after %s...\n", what, time.Since(start).Round(time.Second))
		}
		time.Sleep(doActionPoll)
		next, err := p.client.GetAction(a.ID)
		if err != nil {
			return fmt.Errorf("%s: %w", what, err)
		}
		a = next
	}
}

// dropletAction starts an action on the droplet and waits for it.
func (p *DOProvider) dropletAction(dropletID int, action map[string]string, what string) error {
	a, err := p.client.DropletAction(dropletID, action)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return p.waitForAction(a, what)
}

// CreateImage shuts the droplet down, takes a snapshot named name, and copies
// it to each region of regions that does not have it. A snapshot is private
// to the account.
func (p *DOProvider) CreateImage(meta *SandboxMeta, name string, regions []string) (ImageMeta, error) {
	if meta.DropletID == 0 {
		return ImageMeta{}, ErrNoMachineRecorded
	}
	fmt.Fprintf(os.Stderr, "Shutting down droplet %d...\n", meta.DropletID)
	if err := p.dropletAction(meta.DropletID, map[string]string{"type": "shutdown"}, "the shutdown"); err != nil {
		// A shutdown that the droplet does not answer is a power off.
		fmt.Fprintf(os.Stderr, "Warning: %s; powering the droplet off.\n", err)
		if err := p.dropletAction(meta.DropletID, map[string]string{"type": "power_off"}, "the power off"); err != nil {
			return ImageMeta{}, err
		}
	}
	fmt.Fprintf(os.Stderr, "Taking snapshot %q of droplet %d. This takes several minutes...\n", name, meta.DropletID)
	if err := p.dropletAction(meta.DropletID, map[string]string{"type": "snapshot", "name": name}, "the snapshot"); err != nil {
		return ImageMeta{}, err
	}
	snapshots, err := p.client.ListDropletSnapshots(meta.DropletID)
	if err != nil {
		return ImageMeta{}, fmt.Errorf("finding the snapshot: %w", err)
	}
	var snap *DOImage
	for i := range snapshots {
		if snapshots[i].Name == name {
			snap = &snapshots[i]
		}
	}
	if snap == nil {
		return ImageMeta{}, fmt.Errorf("droplet %d has no snapshot named %q after the snapshot action", meta.DropletID, name)
	}
	img := ImageMeta{ImageID: strconv.Itoa(snap.ID), Regions: slices.Clone(snap.Regions)}
	if len(img.Regions) == 0 {
		img.Regions = []string{meta.Region}
	}
	for _, region := range regions {
		if slices.Contains(img.Regions, region) {
			continue
		}
		fmt.Fprintf(os.Stderr, "Copying snapshot %d to %s...\n", snap.ID, region)
		a, err := p.client.TransferImage(snap.ID, region)
		if err == nil {
			err = p.waitForAction(a, "the copy to "+region)
		}
		if err != nil {
			return img, fmt.Errorf("copying the snapshot to %s: %w", region, err)
		}
		img.Regions = append(img.Regions, region)
	}
	return img, nil
}

// DeleteImage deletes the snapshot in each region that has it.
func (p *DOProvider) DeleteImage(img ImageMeta) error {
	id, err := strconv.Atoi(img.ImageID)
	if err != nil {
		return fmt.Errorf("image ID %q is not a DigitalOcean image ID", img.ImageID)
	}
	if err := p.client.DeleteImage(id); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return fmt.Errorf("%w: %s", ErrImageNotFound, err)
		}
		return err
	}
	return nil
}

// ListImages lists the private images of the account.
func (p *DOProvider) ListImages() ([]ImageMeta, error) {
	images, err := p.client.ListPrivateImages()
	if err != nil {
		return nil, err
	}
	found := make([]ImageMeta, 0, len(images))
	for _, im := range images {
		found = append(found, ImageMeta{Name: im.Name, Provider: p.Name(), ImageID: strconv.Itoa(im.ID), Regions: im.Regions, CreatedAt: im.CreatedAt})
	}
	return found, nil
}
