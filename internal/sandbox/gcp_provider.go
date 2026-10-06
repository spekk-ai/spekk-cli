package sandbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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
		"--image-family", gcpImageFamily,
		"--image-project", gcpImageProject,
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
