package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ProviderGCP is the SandboxMeta.Provider value for a Compute Engine VM.
const ProviderGCP = "gcp"

const (
	// gcpLoginUser is the user spekk logs in as. The image does not admit
	// root over SSH, so the guest agent creates this user, with
	// passwordless sudo, from the ssh-keys metadata.
	gcpLoginUser = "spekk"

	gcpDefaultZone = "us-central1-a"
	// gcpDefaultMachineType has the 2 vCPU and 4 GB of the droplet default.
	gcpDefaultMachineType = "e2-medium"
	gcpImageFamily        = "ubuntu-2404-lts-amd64"
	gcpImageProject       = "ubuntu-os-cloud"
	gcpBootDiskSize       = "50GB"
)

// gcpProjectIDRe is the form of a project ID, with the domain prefix that
// older projects have. gcloud refuses anything else, such as a project's
// display name, before it asks the API, and destroy could then never
// remove the record.
var gcpProjectIDRe = regexp.MustCompile(`^([a-z0-9.-]+:)?[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

// runGcloud is the seam every gcloud call goes through, so a test can run
// the provider without gcloud or a cloud account. It returns trimmed stdout,
// and on failure an error that carries what gcloud wrote to stderr.
var runGcloud = func(args ...string) (string, error) {
	out, err := exec.Command("gcloud", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gcloudNotFound reports whether a gcloud error says the resource does not
// exist.
func gcloudNotFound(err error) bool {
	return strings.Contains(err.Error(), "was not found")
}

// gcloudCannotExist reports whether a gcloud error proves that no VM is at
// the location asked about: it is not found, or the zone or the VM name is
// invalid, so no VM can have it.
func gcloudCannotExist(err error) bool {
	return gcloudNotFound(err) || strings.Contains(err.Error(), "Invalid value for field")
}

// GCPProvider can label its machines and make images of them.
var (
	_ Labeler = (*GCPProvider)(nil)
	_ Imager  = (*GCPProvider)(nil)
)

// gcpImageLabel is the label on each image that spekk makes, so ListImages
// finds them.
const gcpImageLabel = "spekk-image"

// GCPProvider implements Provider for Google Compute Engine through the
// gcloud CLI, which brings the operator's credentials and configuration.
type GCPProvider struct{}

// NewGCPProvider checks that gcloud is installed.
func NewGCPProvider() (*GCPProvider, error) {
	if _, err := exec.LookPath("gcloud"); err != nil {
		return nil, fmt.Errorf("gcloud is not on PATH; install the Google Cloud CLI: https://cloud.google.com/sdk/docs/install")
	}
	return &GCPProvider{}, nil
}

// Name reports the provider name stored in SandboxMeta.Provider.
func (p *GCPProvider) Name() string { return ProviderGCP }

// gcloudConfig returns flag when it is set, and otherwise the operator's
// gcloud configuration value for property, which is empty when unset.
func gcloudConfig(flag, property string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	return runGcloud("config", "get-value", property)
}

// instanceArgs locates the VM for a gcloud compute instances command.
func instanceArgs(verb string, meta *SandboxMeta) []string {
	return []string{"compute", "instances", verb, meta.GCPInstance, "--project", meta.Project, "--zone", meta.Region}
}

// Create makes a VM from the Ubuntu 24.04 image with spekk's cloud-init and a
// generated key, then records the VM name, the resolved project, zone and
// machine type, the address, and the login user on meta.
func (p *GCPProvider) Create(name string, opts CreateOptions, meta *SandboxMeta) error {
	project, err := gcloudConfig(opts.Project, "project")
	if err != nil {
		return fmt.Errorf("reading the gcloud project: %w", err)
	}
	if project == "" {
		return fmt.Errorf("no GCP project: pass --project or run: gcloud config set project <project-id>")
	}
	if !gcpProjectIDRe.MatchString(project) {
		return fmt.Errorf("%q is not a GCP project ID; pass the ID, not the name: gcloud projects list", project)
	}
	zone, err := gcloudConfig(opts.Region, "compute/zone")
	if err != nil {
		return fmt.Errorf("reading the gcloud zone: %w", err)
	}
	if zone == "" {
		zone = gcpDefaultZone
	}
	size := opts.Size
	if size == "" {
		size = gcpDefaultMachineType
	}
	// Destroy finds the VM by these, so record what was used, not what the
	// operator's gcloud configuration says later.
	meta.Project, meta.Region, meta.Size = project, zone, size

	fmt.Fprintln(os.Stderr, "Generating SSH key pair...")
	keyPath, err := generateSSHKeyPair(name)
	if err != nil {
		return fmt.Errorf("generating SSH key: %w", err)
	}
	removeKeys := func() {
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
	}
	pubKeyData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		removeKeys()
		return fmt.Errorf("reading public key: %w", err)
	}
	pubKey := strings.TrimSpace(string(pubKeyData))

	userData, err := os.CreateTemp("", "spekk-user-data-*.yaml")
	if err != nil {
		removeKeys()
		return fmt.Errorf("writing cloud-init: %w", err)
	}
	defer os.Remove(userData.Name())
	_, err = userData.WriteString(renderCloudInit(opts.CloudInit, pubKey))
	if closeErr := userData.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		removeKeys()
		return fmt.Errorf("writing cloud-init: %w", err)
	}

	meta.GCPInstance = "spekk-" + name
	args := instanceArgs("create", meta)
	args = append(args,
		"--machine-type", size,
	)
	if opts.Image != nil {
		imageProject := opts.Image.Project
		if imageProject == "" {
			imageProject = project
		}
		args = append(args, "--image", opts.Image.ImageID, "--image-project", imageProject)
	} else {
		args = append(args, "--image-family", gcpImageFamily, "--image-project", gcpImageProject)
	}
	args = append(args,
		"--boot-disk-size", gcpBootDiskSize,
		// The agent runs code from repositories it clones, so the VM gets
		// no service account: a token from the metadata server would
		// reach the project's resources.
		"--no-service-account", "--no-scopes",
		// enable-oslogin=FALSE keeps a project that turns OS Login on from
		// ignoring the ssh-keys entry, which would lock spekk out.
		// block-project-ssh-keys=TRUE keeps the project's keys off the VM,
		// because the guest agent gives their users sudo, and sudo reads
		// the credentials spekk injects.
		"--metadata", "enable-oslogin=FALSE,block-project-ssh-keys=TRUE,ssh-keys="+gcpLoginUser+":"+pubKey,
		"--metadata-from-file", "user-data="+userData.Name(),
		"--format", "value(networkInterfaces[0].accessConfigs[0].natIP)",
	)
	if opts.VPC != "" {
		args = append(args, "--network", opts.VPC)
	}

	fmt.Fprintf(os.Stderr, "Creating VM %q in %s (%s, project %s)...\n", meta.GCPInstance, zone, size, project)
	ip, createErr := runGcloud(args...)
	if createErr != nil && strings.Contains(createErr.Error(), "already exists") {
		// A VM with this name that spekk did not just make is somebody
		// else's. Recording it would let destroy delete it.
		meta.GCPInstance = ""
		removeKeys()
		return fmt.Errorf("a VM named %s already exists in project %s, zone %s; choose another sandbox name: %w", "spekk-"+name, project, zone, createErr)
	}
	if createErr != nil {
		// gcloud can fail after the insert started, for example while it
		// waits on the operation. Keep the record unless gcloud says no
		// such VM can exist: any other answer leaves a VM that may be
		// billing, and destroy treats a missing VM as done.
		_, err := runGcloud(append(instanceArgs("describe", meta), "--format", "value(name)")...)
		if err != nil && gcloudCannotExist(err) {
			meta.GCPInstance = ""
			removeKeys()
			return fmt.Errorf("creating VM: %w", createErr)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not check whether VM %s exists (%s); keeping its record.\n", meta.GCPInstance, err)
		}
	}
	meta.SSHKeyPath = keyPath
	meta.SSHUser = gcpLoginUser
	if createErr != nil {
		return fmt.Errorf("creating VM: %w", createErr)
	}
	if ip == "" {
		return fmt.Errorf("VM %s has no external IP address", meta.GCPInstance)
	}
	meta.IP = ip
	fmt.Fprintf(os.Stderr, "VM running at %s\n", ip)
	return nil
}

// Destroy deletes the VM and its boot disk. The key is in the VM's metadata, so
// nothing else stays in the cloud.
func (p *GCPProvider) Destroy(meta *SandboxMeta) error {
	if meta.GCPInstance == "" {
		return fmt.Errorf("%w: a VM may still be running and billing", ErrNoMachineRecorded)
	}
	fmt.Fprintf(os.Stderr, "Deleting VM %s...\n", meta.GCPInstance)
	if _, err := runGcloud(append(instanceArgs("delete", meta), "--delete-disks", "boot", "--quiet")...); err != nil {
		if !gcloudNotFound(err) {
			return fmt.Errorf("deleting VM: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Warning: VM %s was already deleted.\n", meta.GCPInstance)
	}
	return nil
}

// Status returns the VM status, for example RUNNING or TERMINATED.
func (p *GCPProvider) Status(meta *SandboxMeta) (string, error) {
	if meta.GCPInstance == "" {
		return "", nil
	}
	return runGcloud(append(instanceArgs("describe", meta), "--format", "value(status)")...)
}

// Label puts the labels on the VM as GCP labels key=value.
func (p *GCPProvider) Label(meta *SandboxMeta, labels map[string]string) error {
	if meta.GCPInstance == "" {
		return fmt.Errorf("no VM recorded")
	}
	_, err := runGcloud(append(instanceArgs("add-labels", meta), "--labels", strings.Join(sortedLabels(labels, "="), ","))...)
	return err
}

// ListLabeled lists the VMs in the default gcloud project that carry a
// spekk-kind label. A VM created before labels existed has none, so it is
// not listed.
func (p *GCPProvider) ListLabeled() ([]RemoteSandbox, error) {
	project, err := gcloudConfig("", "project")
	if err != nil {
		return nil, fmt.Errorf("reading the gcloud project: %w", err)
	}
	if project == "" {
		return nil, fmt.Errorf("no default gcloud project: run gcloud config set project <project-id>")
	}
	out, err := runGcloud("compute", "instances", "list", "--project", project, "--filter", "labels."+labelKind+":*", "--format", "json")
	if err != nil {
		return nil, err
	}
	var vms []struct {
		Name              string            `json:"name"`
		Zone              string            `json:"zone"`
		Labels            map[string]string `json:"labels"`
		CreationTimestamp string            `json:"creationTimestamp"`
		NetworkInterfaces []struct {
			AccessConfigs []struct {
				NatIP string `json:"natIP"`
			} `json:"accessConfigs"`
		} `json:"networkInterfaces"`
	}
	if err := json.Unmarshal([]byte(out), &vms); err != nil {
		return nil, fmt.Errorf("reading the gcloud instance list: %w", err)
	}
	var found []RemoteSandbox
	for _, vm := range vms {
		// The zone is a URL that ends in the zone name.
		zone := vm.Zone[strings.LastIndex(vm.Zone, "/")+1:]
		meta := SandboxMeta{Provider: ProviderGCP, GCPInstance: vm.Name, Project: project, Region: zone}
		if t, err := time.Parse(time.RFC3339, vm.CreationTimestamp); err == nil {
			meta.CreatedAt = t.UTC().Format(time.RFC3339)
		}
		if len(vm.NetworkInterfaces) > 0 && len(vm.NetworkInterfaces[0].AccessConfigs) > 0 {
			meta.IP = vm.NetworkInterfaces[0].AccessConfigs[0].NatIP
		}
		metaFromLabels(&meta, vm.Labels)
		found = append(found, RemoteSandbox{Name: strings.TrimPrefix(vm.Name, "spekk-"), Meta: meta})
	}
	return found, nil
}

// ImageRegion is empty: a GCP custom image is global.
func (p *GCPProvider) ImageRegion(CreateOptions) string { return "" }

// CreateImage stops the VM and makes a custom image of its boot disk, which
// has the VM's name. The image is global and private to the project.
func (p *GCPProvider) CreateImage(meta *SandboxMeta, name string, _ []string) (ImageMeta, error) {
	if meta.GCPInstance == "" {
		return ImageMeta{}, ErrNoMachineRecorded
	}
	fmt.Fprintf(os.Stderr, "Stopping VM %s...\n", meta.GCPInstance)
	if _, err := runGcloud(instanceArgs("stop", meta)...); err != nil {
		return ImageMeta{}, fmt.Errorf("stopping VM: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Making image %q from the boot disk of %s. This takes several minutes...\n", name, meta.GCPInstance)
	if _, err := runGcloud("compute", "images", "create", name, "--project", meta.Project,
		"--source-disk", meta.GCPInstance, "--source-disk-zone", meta.Region,
		"--labels", gcpImageLabel+"=true", "--format", "value(name)"); err != nil {
		return ImageMeta{}, fmt.Errorf("making the image: %w", err)
	}
	return ImageMeta{ImageID: name, Project: meta.Project}, nil
}

// DeleteImage deletes the custom image.
func (p *GCPProvider) DeleteImage(img ImageMeta) error {
	if _, err := runGcloud("compute", "images", "delete", img.ImageID, "--project", img.Project, "--quiet"); err != nil {
		if gcloudNotFound(err) {
			return fmt.Errorf("%w: %s", ErrImageNotFound, err)
		}
		return err
	}
	return nil
}

// ListImages lists the images with the spekk-image label in the default
// gcloud project.
func (p *GCPProvider) ListImages() ([]ImageMeta, error) {
	project, err := gcloudConfig("", "project")
	if err != nil {
		return nil, fmt.Errorf("reading the gcloud project: %w", err)
	}
	if project == "" {
		return nil, fmt.Errorf("no default gcloud project: run gcloud config set project <project-id>")
	}
	out, err := runGcloud("compute", "images", "list", "--project", project, "--no-standard-images",
		"--filter", "labels."+gcpImageLabel+":*", "--format", "json")
	if err != nil {
		return nil, err
	}
	var images []struct {
		Name              string `json:"name"`
		CreationTimestamp string `json:"creationTimestamp"`
	}
	if err := json.Unmarshal([]byte(out), &images); err != nil {
		return nil, fmt.Errorf("reading the gcloud image list: %w", err)
	}
	found := make([]ImageMeta, 0, len(images))
	for _, im := range images {
		img := ImageMeta{Name: im.Name, Provider: ProviderGCP, ImageID: im.Name, Project: project}
		if t, err := time.Parse(time.RFC3339, im.CreationTimestamp); err == nil {
			img.CreatedAt = t.UTC().Format(time.RFC3339)
		}
		found = append(found, img)
	}
	return found, nil
}
