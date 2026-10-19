package sandbox

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/config"
	"github.com/spekk-ai/spekk-cli/internal/version"
)

var sandboxNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidateSandboxName checks that a sandbox name contains only safe characters.
// Valid names match ^[a-z0-9][a-z0-9-]*$ (lowercase alphanumeric and hyphens,
// must start with alphanumeric). This prevents shell injection when the name
// is used in commands or environment variables.
func ValidateSandboxName(name string) error {
	if name == "" {
		return fmt.Errorf("sandbox name cannot be empty")
	}
	if !sandboxNameRe.MatchString(name) {
		return fmt.Errorf("invalid sandbox name %q: must match [a-z0-9][a-z0-9-]* (lowercase alphanumeric and hyphens, starting with alphanumeric)", name)
	}
	return nil
}

// CreateOptions holds flags for the create subcommand.
type CreateOptions struct {
	Name    string
	Region  string
	Size    string
	Project string
	VPC     string
	IP      string // address of a machine that already exists
	SSHKey  string // private key that reaches that machine as the login user
	SSHUser string // login user for an existing machine (default: root)
	Auth    AuthMode

	// Organization is the control host organization the agent is registered
	// in. Empty lets the control host choose, which works for a user who
	// belongs to one organization.
	Organization string

	// Release is the spekk release the cloud-init template and agent binary
	// come from. Empty means the latest published release; a specific tag
	// (e.g. an "exp-*" prerelease) pins a build that "latest" does not carry.
	Release string

	// ProvisionTimeout is how long Create waits for cloud-init to write the
	// provisioned marker. Zero means DefaultProvisionTimeout.
	ProvisionTimeout time.Duration

	// Kind is KindOneOff or KindPersistent; empty means persistent.
	// ExpiresAt is when a one-off sandbox expires; zero means never.
	// Purpose says why the sandbox exists. ResolveKind fills Kind and
	// ExpiresAt from the flags.
	Kind      string
	ExpiresAt time.Time
	Purpose   string

	// Labels are extra labels for the machine, from --label in machine mode.
	Labels map[string]string

	// ImageName is the image of --image. Image is that image when the
	// provider has it in the region of the create; Create resolves it, and
	// a provider that supports images creates the machine from it.
	ImageName string
	Image     *ImageMeta

	// CloudInit is the provisioning payload from the release artifacts.
	// Create fills it in; no flag sets it. A provider that does not use
	// cloud-init ignores it.
	CloudInit []byte
}

// ProvisionOptions holds flags for the provision subcommand.
type ProvisionOptions struct {
	// Auth overrides the mode the record carries. Empty keeps the recorded
	// mode, which is bedrock for a record written before the field existed.
	Auth AuthMode
	// Force provisions a record whose status is not "provisioning".
	Force bool
	// Organization is used only when provision registers a new agent. See
	// CreateOptions.Organization.
	Organization string
	// Release pins the spekk release the agent binary comes from. Empty means
	// the latest published release. See CreateOptions.Release.
	Release string
}

// releaseTag is the release the artifacts come from: the tag the operator
// pinned with --release, else the release of this CLI, so the operator and
// the sandbox match. A build that is not a release gets the latest release.
func releaseTag(pinned string) string {
	if pinned != "" {
		return pinned
	}
	if releaseVersionRe.MatchString(version.Version) {
		return version.Version
	}
	return "latest"
}

// fetchRelease fetches the artifacts of the release that releaseTag picks,
// and says so when that is the latest release and not the one of this CLI.
func fetchRelease(pinned string) (*releaseArtifacts, error) {
	fmt.Fprintln(os.Stderr, "Fetching sandbox release artifacts...")
	tag := releaseTag(pinned)
	artifacts, err := fetchArtifacts(tag)
	if err != nil {
		return nil, fmt.Errorf("fetching release artifacts: %w", err)
	}
	if tag == "latest" {
		fmt.Fprintf(os.Stderr, "This spekk is %q, not a release, so the sandbox gets the latest release, %s.\n", version.Version, artifacts.Version)
	}
	return artifacts, nil
}

// checkRequiredEnv reports every variable the auth mode needs that is not
// set, so a subscription sandbox is not blocked by AWS keys it will never
// use. Create and Provision both call it before touching a machine.
func checkRequiredEnv(mode AuthMode) error {
	var missing []string
	for _, v := range requiredEnvVars(mode) {
		if strings.TrimSpace(os.Getenv(v)) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if mode == AuthSubscription && os.Getenv(codexAuthFileVar) != "" {
		if _, err := readCodexAuth(os.Getenv(codexAuthFileVar)); err != nil {
			return err
		}
	}
	return nil
}

// --- Create ---

// Create provisions a sandbox using the given Provider for VM lifecycle and
// then runs generic provisioning (SSH wait, credential injection, agent deploy).
func Create(p Provider, opts CreateOptions) (err error) {
	if err := checkRequiredEnv(opts.Auth); err != nil {
		return err
	}

	// A name that is already taken is nearly always a create that failed
	// partway. Overwriting its entry would drop the identifier of the
	// machine that failure left running.
	if existing, err := GetSandbox(opts.Name); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("sandbox %q already exists (%s); destroy it first: spekk sandbox destroy %s",
			opts.Name, machineRef(existing), opts.Name)
	}

	// Fetch release artifacts before creating billable resources.
	artifacts, err := fetchRelease(opts.Release)
	if err != nil {
		return err
	}
	// The binaries are downloaded later (arch is not known yet), so defer over them.
	defer func() { artifacts.cleanup() }()
	fmt.Fprintf(os.Stderr, "Using sandbox release %s\n", artifacts.Version)

	opts.CloudInit = artifacts.CloudInit
	if err := applyImage(p, &opts, time.Now()); err != nil {
		return err
	}

	// Register before anything billable exists, so a refused registration
	// costs nothing. From here on, a failure revokes the agent again.
	agentToken, agentID, err := newAgentToken(opts.Name, opts.Organization)
	if err != nil {
		return err
	}

	// Delegate machine creation to the provider, which fills in the fields
	// of meta that it owns. A nil provider means no cloud owns this
	// machine: it already exists, and spekk only registers and equips it.
	meta := &SandboxMeta{
		Provider:  ProviderNone,
		Auth:      string(opts.Auth),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AgentID:   agentID,
		Kind:      opts.Kind,
		Purpose:   opts.Purpose,
		CreatedBy: hostname(),
	}
	if meta.Kind == "" {
		meta.Kind = KindPersistent
	}
	if !opts.ExpiresAt.IsZero() {
		meta.ExpiresAt = opts.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if opts.Image != nil {
		meta.Image = opts.Image.Name
	}
	defer func() {
		if err != nil && meta.AgentID != "" && revokeAbandoned(meta.AgentID, opts.Name) {
			meta.AgentID = ""
			if namesMachine(meta) {
				if saveErr := SaveSandbox(opts.Name, meta); saveErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not record that the agent is revoked: %s\n", saveErr)
				}
			}
		}
	}()
	if agentID != "" {
		// Deferred after the revoke above, so it runs first: once the
		// watch stops, only the normal path decides about the agent.
		defer revokeOnSignal(agentID, opts.Name)()
	}
	if p != nil {
		meta.Provider = p.Name()
	}
	// Record the machine as soon as one exists, and record it even when
	// the provider then fails. A machine with no metadata entry is one
	// that `spekk sandbox destroy` cannot reach: the operator has to go
	// to the provider's console and match it by name.
	meta.Status = "provisioning"
	var createErr error
	if p == nil {
		createErr = registerMachine(opts, meta)
	} else {
		createErr = p.Create(opts.Name, opts, meta)
	}
	if namesMachine(meta) {
		if err := SaveSandbox(opts.Name, meta); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not record %s: %s\n", machineRef(meta), err)
			if createErr == nil {
				return fmt.Errorf("saving metadata for %s: %w", machineRef(meta), err)
			}
		}
	}
	if createErr != nil {
		if namesMachine(meta) {
			fmt.Fprintf(os.Stderr, "%s -- not auto-destroyed. Run: spekk sandbox destroy %s\n", machineRef(meta), opts.Name)
		}
		return createErr
	}
	labelMachine(p, meta)

	// --- Generic provisioning (provider-agnostic) ---

	// A failure from here on leaves a real machine running, and a record
	// at "provisioning". Say so, say which machine, and say how to finish:
	// `spekk sandbox provision` picks up from here, so a slow cloud-init
	// does not cost the operator the droplet.
	fail := func(stage string, err error) error {
		fmt.Fprintf(os.Stderr, "%s -- not auto-destroyed. Finish it with: spekk sandbox provision %s\nOr remove it with: spekk sandbox destroy %s\n",
			machineRef(meta), opts.Name, opts.Name)
		return fmt.Errorf("%s: %w", stage, err)
	}

	if p == nil {
		// Nothing to wait for: the operator provisioned this machine.
		// Confirm that before spending credentials on it.
		if err := checkReady(meta, opts.Name); err != nil {
			return fail("checking the machine", err)
		}
	} else if err := waitReady(meta.IP, meta.SSHKeyPath, opts.Name, sshUser(meta), opts.ProvisionTimeout); err != nil {
		return fail("waiting for provisioning", err)
	}
	fmt.Fprintln(os.Stderr, "Provisioning complete.")

	if err := equip(meta, opts.Name, agentToken, opts.Auth, artifacts); err != nil {
		return fail(err.stage, err.err)
	}

	meta.Status = "active"
	if err := SaveSandbox(opts.Name, meta); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	printResult("Sandbox created successfully", opts.Name, meta, agentToken)
	return nil
}

// equip is the seam Create and Provision equip through, so a test can reach
// the steps around it without an SSH connection.
var equip = equipSandbox

// stageError is a failure from one step of equipSandbox, with the step it
// came from, so Create and Provision can say which step stopped them.
type stageError struct {
	stage string
	err   error
}

func (e *stageError) Error() string { return e.stage + ": " + e.err.Error() }
func (e *stageError) Unwrap() error { return e.err }

// equipSandbox runs the steps that turn a provisioned machine into a sandbox:
// inject the credentials, configure git for the agent, deploy the agent
// binary. Create runs it after the provisioning wait, and Provision runs it
// on a record that wait left behind, so the two cannot drift apart.
func equipSandbox(meta *SandboxMeta, name, agentToken string, mode AuthMode, artifacts *releaseArtifacts) *stageError {
	// The login user for every SSH step. root on a machine spekk created;
	// whatever the operator gave for one they already had.
	user := sshUser(meta)

	if err := fetchAgent(meta, name, artifacts); err != nil {
		return &stageError{"fetching agent binary", err}
	}

	fmt.Fprintln(os.Stderr, "Injecting credentials...")
	if err := injectCredentials(meta.IP, meta.SSHKeyPath, name, user, agentToken, mode, markerContent(meta)); err != nil {
		return &stageError{"injecting credentials", err}
	}

	fmt.Fprintln(os.Stderr, "Configuring git credentials...")
	if err := configureGitCredentials(meta.IP, meta.SSHKeyPath, name, user); err != nil {
		return &stageError{"configuring git credentials", err}
	}

	fmt.Fprintln(os.Stderr, "Deploying the agent and the spekk CLI...")
	if err := deployAgent(meta.IP, meta.SSHKeyPath, name, user, artifacts); err != nil {
		return &stageError{"deploying agent", err}
	}
	return nil
}

// fetchAgent is the seam equipSandbox fetches the agent build through, so a
// test can reach the credential step without an SSH connection.
var fetchAgent = fetchAgentBinary

// fetchAgentBinary picks the agent build for the machine's own CPU, so it runs
// only after create/provision has reached the machine over SSH.
func fetchAgentBinary(meta *SandboxMeta, name string, artifacts *releaseArtifacts) error {
	arch, err := detectArch(meta, name)
	if err != nil {
		return err
	}
	return artifacts.downloadBinaries(arch)
}

// detectArch reads `uname -m` from stdout only. ssh writes its own warnings
// to stderr, for example about a key exchange that is not post-quantum, and
// those lines would otherwise reach archFromUname as the machine name.
func detectArch(meta *SandboxMeta, name string) (string, error) {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if meta.SSHKeyPath != "" {
		args = append(args, "-i", meta.SSHKeyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", sshUser(meta), meta.IP), "uname -m")
	out, err := exec.Command("ssh", args...).Output()
	if err != nil {
		var stderr []byte
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = exitErr.Stderr
		}
		return "", fmt.Errorf("detecting CPU architecture: %w\n%s", err, strings.TrimSpace(string(stderr)))
	}
	return archFromUname(strings.TrimSpace(string(out)))
}

// archFromUname maps `uname -m` to the GOARCH the agent is built for. An arch
// spekk publishes no agent for is refused, not guessed: the alternative is
// deploying a binary the machine cannot run and reporting success.
func archFromUname(machine string) (string, error) {
	switch machine {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported CPU architecture %q: spekk publishes an agent for x86_64 and aarch64 only", machine)
	}
}

// printResult reports a finished sandbox. A registered sandbox needs nothing
// more, and its token stays off the terminal. An unregistered one prints the
// token for the manual path, as create always did.
func printResult(headline, name string, meta *SandboxMeta, agentToken string) {
	if meta.AgentID != "" {
		fmt.Fprintf(os.Stderr, "\n%s:\n  Name:   spekk-%s\n  IP:     %s\n  Agent:  %s (registered on the control host)\n", headline, name, meta.IP, meta.AgentID)
		return
	}
	printRegistration(headline, name, meta.IP, agentToken)
}

// printRegistration prints the token the operator has to register on the
// control host. The agent cannot connect until they do, so the message is
// the same whichever command finished the sandbox.
func printRegistration(headline, name, ip, agentToken string) {
	host := bareHost(os.Getenv("SPEKK_HOST"))

	fmt.Fprintf(os.Stderr, `
%s:
  Name:           spekk-%s
  IP:             %s
  AGENT_TOKEN:    %s

Next: Register this agent on the control host admin at https://%s/
  - Name: %s
  - Sandbox ID: spekk-%s
  - Auth token: %s

Or register it with your spekk API token, with no admin step:
  export %s=<your API token>
  spekk sandbox register %s
`, headline, name, ip, agentToken, host, name, name, agentToken, apiTokenVar, name)
}

// --- Provision ---

// Provision finishes a sandbox that Create left at "provisioning": the
// machine exists and cloud-init may have finished after Create stopped
// waiting, but no credentials or agent reached it. It checks the marker,
// then runs the same steps Create runs after its wait, and records the
// sandbox as active.
func Provision(name string, opts ProvisionOptions) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	// A record that is not provisioning has already been equipped, or was
	// never a machine spekk should touch. --force says the operator knows.
	if sandbox.Status != "provisioning" && !opts.Force {
		return fmt.Errorf("sandbox %q is %s, not provisioning; use --force to provision it again", name, orUnknown(sandbox.Status))
	}

	mode := opts.Auth
	if mode == "" {
		if mode, err = ParseAuthMode(sandbox.Auth); err != nil {
			return fmt.Errorf("recorded auth mode: %w", err)
		}
	}

	// Fail before touching the machine when a credential is missing, for
	// the same reason Create does: a half-written env file is worse than
	// none.
	if err := checkRequiredEnv(mode); err != nil {
		return err
	}

	if err := checkReady(sandbox, name); err != nil {
		return fmt.Errorf("checking the machine: %w", err)
	}

	artifacts, err := fetchRelease(opts.Release)
	if err != nil {
		return err
	}
	// The binaries are downloaded later (arch is not known yet), so defer over them.
	defer func() { artifacts.cleanup() }()

	// A registered sandbox keeps its agent: a new token would disconnect
	// it, and a new agent would lose its schedules and its history. Only a
	// sandbox with no agent gets a new token, registered when it can be.
	var agentToken, agentID string
	if sandbox.AgentID != "" {
		if agentToken, err = readAgentToken(sandbox, name); err != nil {
			return fmt.Errorf("reading the agent token on %s: %w\nRegister a new agent with: spekk sandbox register %s", machineRef(sandbox), err, name)
		}
		agentID = sandbox.AgentID
	} else if agentToken, agentID, err = newAgentToken(name, opts.Organization); err != nil {
		return err
	}
	if err := equip(sandbox, name, agentToken, mode, artifacts); err != nil {
		if sandbox.AgentID == "" && agentID != "" {
			revokeAbandoned(agentID, name)
		}
		return err
	}

	registered := sandbox.AgentID == "" && agentID != ""
	sandbox.Auth = string(mode)
	sandbox.Status = "active"
	sandbox.AgentID = agentID
	if err := SaveSandbox(name, sandbox); err != nil {
		if registered {
			// The machine runs the new agent, so revoking it would be
			// wrong. Name it, because no record does.
			return fmt.Errorf("saving metadata: %w (agent %s is registered and running on the machine)", err, agentID)
		}
		return fmt.Errorf("saving metadata: %w", err)
	}

	printResult("Sandbox provisioned successfully", name, sandbox, agentToken)
	return nil
}

// --- List ---

// ListOptions holds flags for the list subcommand.
type ListOptions struct {
	// Expired shows only the sandboxes whose expiry is in the past.
	Expired bool
	// AllHosts also lists the labeled machines that the providers report
	// and no local record names, for example those another machine made.
	AllHosts bool
}

// listRow is one sandbox in the list table.
type listRow struct {
	name string
	meta *SandboxMeta
}

// List displays the sandboxes in a table, sorted by name.
func List(opts ListOptions) error {
	return list(os.Stdout, os.Stderr, opts, time.Now())
}

func list(out, errOut io.Writer, opts ListOptions, now time.Time) error {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return err
	}
	var all []listRow
	for name, meta := range sandboxes {
		all = append(all, listRow{name, meta})
	}
	if opts.AllHosts {
		all = append(all, remoteRows(sandboxes, errOut)...)
	}
	var rows []listRow
	for _, r := range all {
		if !opts.Expired || r.meta.Expired(now) {
			rows = append(rows, r)
		}
	}
	if len(rows) == 0 {
		if opts.Expired {
			fmt.Fprintln(out, "No expired sandboxes found.")
		} else {
			fmt.Fprintln(out, "No sandboxes found.")
		}
		return nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

	header := padRow("Name", "Kind", "IP", "Region", "Status", "Created", "Expires")
	fmt.Fprintln(out, header)
	fmt.Fprintln(out, strings.Repeat("-", len(header)))
	for _, r := range rows {
		fmt.Fprintln(out, padRow(
			r.name,
			r.meta.KindOf(),
			orDash(r.meta.IP),
			orDash(r.meta.Region),
			orDash(r.meta.Status),
			orDash(r.meta.CreatedAt),
			expiresCell(r.meta, now),
		))
	}
	return nil
}

// expiresCell is the Expires column: the expiry, marked when it is past.
func expiresCell(meta *SandboxMeta, now time.Time) string {
	if meta.Expired(now) {
		return "EXPIRED " + meta.ExpiresAt
	}
	return orDash(meta.ExpiresAt)
}

// --- Status ---

// Status shows detailed info for a named sandbox, using the Provider
// to fetch live VM state.
func Status(p Provider, name string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	fmt.Printf("Sandbox: %s\n", name)
	fmt.Printf("Provider: %s\n", orUnknown(providerName(sandbox)))
	if sandbox.DropletID != 0 {
		fmt.Printf("Droplet ID: %d\n", sandbox.DropletID)
	}
	if sandbox.GCPInstance != "" {
		fmt.Printf("VM: %s\n", sandbox.GCPInstance)
	}
	fmt.Printf("IP: %s\n", orUnknown(sandbox.IP))
	fmt.Printf("Region: %s\n", orUnknown(sandbox.Region))
	fmt.Printf("Size: %s\n", orUnknown(sandbox.Size))
	fmt.Printf("Created: %s\n", orUnknown(sandbox.CreatedAt))
	fmt.Printf("Kind: %s\n", sandbox.KindOf())
	fmt.Printf("Purpose: %s\n", orDash(sandbox.Purpose))
	fmt.Printf("Expires: %s\n", expiresCell(sandbox, time.Now()))
	fmt.Printf("Created by: %s\n", orDash(sandbox.CreatedBy))

	// Fetch live status from the provider. A nil provider has two causes:
	// no cloud owns this machine, which is normal and silent, or one could
	// not be built — most often no API token — which is worth saying. Both
	// fall back to the stored value rather than refusing to print.
	vmStatus := orUnknown(sandbox.Status) + " (stored)"
	if p == nil {
		if sandbox.Provider != ProviderNone {
			fmt.Fprintln(os.Stderr, "Warning: no provider available; showing the stored status.")
		}
	} else if live, err := p.Status(sandbox); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch live VM status: %s\n", err)
	} else if live != "" {
		vmStatus = live
	}
	fmt.Printf("VM status: %s\n", vmStatus)

	// SSH checks
	provisioned := sshCheck(sandbox, name, "test -f /opt/spekk/.provisioned && echo yes || echo no")
	fmt.Printf("Provisioned: %s\n", provisioned)

	agentStatus := sshCheck(sandbox, name, "systemctl is-active spekk-agent 2>/dev/null || echo unknown")
	fmt.Printf("Agent service: %s\n", agentStatus)

	printBinaryVersions(sandbox, name)
	return nil
}

// versionCommand prints the version of a binary alone, or "missing" when
// the binary is not there or prints no version. An old agent-client prints
// "version: dev". The binary runs as agent, which owns it and can replace
// it, so the login user (root, or a user with sudo) never runs it.
func versionCommand(user, run, filter string) string {
	asAgent := "sudo -n -u agent "
	if user == "root" {
		asAgent = "runuser -u agent -- "
	}
	return asAgent + run + " 2>/dev/null" + filter + " | grep . || echo missing"
}

// printBinaryVersions prints the version of each spekk binary on the
// machine, with a warning when it is not the release that a deploy with no
// --release installs.
func printBinaryVersions(sandbox *SandboxMeta, name string) {
	want := releaseTag("")
	wantName := want + ", the release of this spekk"
	if want == "latest" {
		latest, err := resolveReleaseTag(want)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: versions not compared with the latest release: %s\n", err)
			want, wantName = "", ""
		} else {
			want, wantName = latest, latest+", the latest release"
		}
	}
	user := sshUser(sandbox)
	for _, b := range []struct{ label, command string }{
		{"spekk CLI", versionCommand(user, "/opt/spekk/cli/spekk version", "")},
		{"Agent client", versionCommand(user, "/opt/spekk/agent-client --version", ` | sed -n 's/^version: //p'`)},
	} {
		got := sshCheck(sandbox, name, b.command)
		line := fmt.Sprintf("%s: %s", b.label, got)
		if reason := versionDrift(got, want, wantName); reason != "" {
			line += fmt.Sprintf("  WARNING: %s; update with: spekk sandbox deploy %s", reason, name)
		}
		fmt.Println(line)
	}
}

// versionDrift is why a binary version read on a machine needs a deploy,
// or "" when it does not. want is the expected release, and wantName says
// what it is; an empty want compares nothing. "unknown" is no SSH answer,
// which the other checks of status already show.
func versionDrift(got, want, wantName string) string {
	switch {
	case got == "unknown":
		return ""
	case got == "missing":
		return "not installed, or no version"
	case got == "dev":
		return "a dev build, with no release stamp"
	case want != "" && got != want:
		return "not " + wantName
	}
	return ""
}

// --- SSH ---

// SSH opens an interactive SSH session to a sandbox.
func SSH(name string, extraArgs []string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	args := sshArgs(sandbox, name)
	args = append(args, extraArgs...)
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("SSH failed: %w", err)
	}
	return nil
}

// --- Destroy ---

// DestroyOptions holds the flags of `spekk sandbox destroy`.
type DestroyOptions struct {
	// Force skips the confirmation, and removes a record whose machine
	// teardown failed or that names no machine.
	Force bool
	// IgnoreRevokeFailure continues when the agent cannot be revoked on the
	// control host. Force does not, because a live agent with no machine
	// is a credential that nothing points at.
	IgnoreRevokeFailure bool
	// CallerRevokes skips the revoke, because the caller revokes the agent
	// itself. Machine mode sets it: it has no API token, and the server
	// that runs it owns the agent.
	CallerRevokes bool
}

// Destroy tears down a sandbox using the Provider for remote resource cleanup
// and then removes local files and metadata.
func Destroy(p Provider, name string, opts DestroyOptions) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	// Without SPEKK_API_TOKEN or SPEKK_HOST the revoke cannot even start.
	// Say so before the prompt, not after the user has said yes.
	revoke := sandbox.AgentID != "" && !opts.CallerRevokes
	if revoke && !opts.IgnoreRevokeFailure {
		if _, err := newControlHost(); err != nil {
			return fmt.Errorf("cannot revoke agent %s of sandbox %q: %w\nNothing was destroyed. Use --ignore-revoke-failure to destroy the sandbox and leave the agent live", sandbox.AgentID, name, err)
		}
	}

	if !opts.Force {
		fmt.Fprintf(os.Stderr, "Destroy sandbox %q (%s)? [y/N] ", name, machineRef(sandbox))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Revoke first. A machine that is gone with its agent still live on the
	// control host leaves a credential that nothing points at any more.
	if revoke {
		if err := revokeRecordedAgent(sandbox.AgentID); err != nil {
			if !opts.IgnoreRevokeFailure {
				return fmt.Errorf("revoking agent %s on the control host: %w\nNothing was destroyed. Use --ignore-revoke-failure to destroy the sandbox anyway", sandbox.AgentID, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: could not revoke agent %s on the control host: %s; destroying anyway because --ignore-revoke-failure was given. Revoke it on the control host.\n", sandbox.AgentID, err)
		} else {
			fmt.Fprintf(os.Stderr, "Revoked agent %s on the control host.\n", sandbox.AgentID)
		}
	}

	// A manual machine survives destroy, so stopping the agent and
	// removing its credentials is the whole of teardown. Failing to do it
	// must stop the command: deleting the local record would leave an
	// agent running with live credentials and nothing pointing at it.
	// A cloud machine is about to be deleted whole, so this does not apply.
	if sandbox.Provider == ProviderNone {
		if err := stopAgent(sandbox, name); err != nil {
			if !opts.Force {
				return fmt.Errorf("stopping the agent on %s: %w\nThe machine may still be running it. Use --force to remove the record anyway", machineRef(sandbox), err)
			}
			fmt.Fprintf(os.Stderr, "Warning: could not stop the agent on %s: %s; removing the local record anyway because --force was given.\n", machineRef(sandbox), err)
		}
	}

	// Delegate remote resource cleanup to the provider. This is not
	// optional: if it fails, stop, because removing the metadata below is
	// what makes an orphaned machine unfindable. A nil provider means no
	// cloud owns the machine, so there is nothing of its to tear down.
	if err := destroyMachine(p, sandbox); err != nil {
		// A record that names no machine is normally a lost identifier,
		// and removing it would hide a running machine. With --force the
		// operator has said they checked; let them clear the entry.
		if !errors.Is(err, ErrNoMachineRecorded) || !opts.Force {
			return err
		}
		fmt.Fprintf(os.Stderr, "Warning: %s; removing the local record anyway because --force was given.\n", err)
	}

	removeGeneratedKeyPair(sandbox.SSHKeyPath)

	// Remove per-sandbox known_hosts file.
	os.Remove(KnownHostsFile(name))

	if err := RemoveSandbox(name); err != nil {
		return fmt.Errorf("removing metadata: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Sandbox %q destroyed.\n", name)
	return nil
}

// removeGeneratedKeyPair deletes the local key pair at path, but only when
// spekk generated it; ownsKeyPair holds the rule. An operator-supplied key
// belongs to the operator, and saying that it stayed is what tells them the
// destroy did not take it.
func removeGeneratedKeyPair(path string) {
	if path == "" {
		return
	}
	if !ownsKeyPair(path) {
		fmt.Fprintf(os.Stderr, "Kept SSH key %s: spekk did not generate it.\n", path)
		return
	}
	os.Remove(path)
	os.Remove(path + ".pub")
}

// providerName reports the provider that owns a sandbox, reading an entry
// written before the field existed as DigitalOcean.
func providerName(meta *SandboxMeta) string {
	if meta.Provider == "" {
		return "digitalocean"
	}
	return meta.Provider
}

// destroyMachine tears down the provider's resources, if a provider owns any.
func destroyMachine(p Provider, meta *SandboxMeta) error {
	if p == nil {
		return nil
	}
	return p.Destroy(meta)
}

// namesMachine reports whether meta identifies a machine that exists.
func namesMachine(meta *SandboxMeta) bool {
	return meta.DropletID != 0 || meta.GCPInstance != "" || meta.IP != ""
}

// machineRef describes the machine a command is about to act on, so the
// operator can check it against the provider's console before saying yes.
func machineRef(meta *SandboxMeta) string {
	ref := "IP " + orUnknown(meta.IP)
	if meta.DropletID != 0 {
		ref += fmt.Sprintf(", droplet %d", meta.DropletID)
	}
	if meta.GCPInstance != "" {
		ref += fmt.Sprintf(", VM %s in %s", meta.GCPInstance, meta.Region)
	}
	return ref
}

// --- Deploy ---

// Deploy downloads and deploys the agent binary and the spekk CLI to a
// sandbox. release pins the spekk release to pull from; empty means the one
// releaseTag picks.
func Deploy(name, release string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	fmt.Fprintf(os.Stderr, "Deploying agent to %s...\n", sandbox.IP)
	artifacts, err := fetchRelease(release)
	if err != nil {
		return err
	}
	// The binaries are downloaded later (arch is not known yet), so defer over them.
	defer func() { artifacts.cleanup() }()

	if err := fetchAgentBinary(sandbox, name, artifacts); err != nil {
		return fmt.Errorf("fetching agent binary: %w", err)
	}
	if err := deployAgent(sandbox.IP, sandbox.SSHKeyPath, name, sshUser(sandbox), artifacts); err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Agent and spekk CLI %s deployed to %q.\n", artifacts.Version, name)
	return nil
}

// spekkAgentUnit is the systemd unit installed on each sandbox. It is the single
// source of truth for the service definition (matches the release cloud-init).
const spekkAgentUnit = `[Unit]
Description=Spekk Agent Client
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=agent
WorkingDirectory=/opt/spekk
EnvironmentFile=/etc/spekk/agent.env
ExecStart=/opt/spekk/agent-client
Restart=always
RestartSec=5
StandardOutput=append:/var/log/spekk/agent.log
StandardError=append:/var/log/spekk/agent.log

[Install]
WantedBy=multi-user.target
`

// deployAgent copies the agent binary to /opt/spekk/agent-client and the
// spekk CLI to /opt/spekk/cli/spekk, upgrades Claude Code when it is older
// than the minimum, installs the systemd unit, and (re)starts the service.
// Shared by Create and Deploy.
func deployAgent(ip, keyPath, name, user string, artifacts *releaseArtifacts) error {
	// Upload into the login user's home so SCP cannot overwrite the running executable.
	scp := sshHostKeyOpts(name)
	scp = append(scp, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		scp = append(scp, "-i", keyPath)
	}
	scp = append(scp, artifacts.BinaryPath, artifacts.CLIPath, scpTarget(user, ip))
	if out, err := scpExec(scp); err != nil {
		return fmt.Errorf("copying binary: %s\n%s", err, string(out))
	}

	// Before the restart: a new agent refuses every turn on an older Claude
	// Code, so a deploy that cannot upgrade it keeps the agent that runs.
	if err := ensureClaudeCode(ip, keyPath, name, user); err != nil {
		return err
	}
	if err := ensureCodexCLI(ip, keyPath, name, user); err != nil {
		return err
	}

	// Install the unit, fix ownership, and start the service. The agent needs to
	// own /opt/spekk so it can create its WORKSPACE (/opt/spekk/workspace) at runtime.
	script := fmt.Sprintf(`set -e
mkdir -p /opt/spekk/workspace
chown -R agent:agent /opt/spekk
mkdir -p /var/log/spekk
chown agent:agent /var/log/spekk
cat > /etc/systemd/system/spekk-agent.service << 'UNITEOF'
%s
UNITEOF
systemctl daemon-reload
systemctl enable spekk-agent
systemctl restart spekk-agent`, spekkAgentUnit)

	out, err := runSSHCombined(ip, keyPath, name, user, installCommand(user, artifacts.AgentSHA256, artifacts.CLISHA256, script))
	if err != nil {
		return fmt.Errorf("installing service: %s\n%s", err, out)
	}
	// The install script says "spekk: ..." for a step that it skipped.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "spekk: ") {
			fmt.Fprintln(os.Stderr, line)
		}
	}
	return nil
}

// stagedBinary and stagedCLI are the upload paths relative to the SSH login
// user's home.
const (
	stagedBinary = "agent-client.staged"
	stagedCLI    = "spekk.staged"
)

// sshExec runs an ssh command, with stdin as its standard input when it is
// not empty. It is a variable so a test can read what the privileged steps
// actually send, with no machine to send it to.
var sshExec = func(args []string, stdin string) ([]byte, error) {
	cmd := exec.Command("ssh", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	return cmd.CombinedOutput()
}

// scpExec copies a file with scp. It is a seam for the same reason sshExec
// is: the deploy decides where a non-root login stages the agent binary, and
// that decision is worth a test.
var scpExec = func(args []string) ([]byte, error) {
	return exec.Command("scp", args...).CombinedOutput()
}

// privilegedScript runs script directly for root and through sudo for other users.
// shellArgs are quoted shell expressions expanded by the login shell before escalation.
func privilegedScript(user, script string, shellArgs ...string) string {
	if user == "root" {
		if len(shellArgs) > 0 {
			return "set -- " + strings.Join(shellArgs, " ") + "\n" + script
		}
		return script
	}
	return sudoWrap(script, shellArgs...)
}

// scpTarget stages each upload in the SSH login user's home. The uploads
// keep their file names, which are the staged names.
func scpTarget(user, ip string) string {
	return fmt.Sprintf("%s@%s:", user, ip)
}

// installCommand checks the uploads against the release sha256, replaces
// the agent and the CLI, and then runs the service setup script.
func installCommand(user, agentSHA256, cliSHA256, script string) string {
	// The check comes before any change, so a bad file replaces nothing.
	// Prepare on the destination filesystem so the final rename is atomic.
	// The directory comes first: a machine an operator provisioned by hand may
	// not carry it, and every later step writes inside it. The CLI belongs to
	// agent, so `spekk update` as agent can replace it. A copy in agent's
	// ~/.local/bin, which an earlier install made, becomes a link to it, so
	// what calls that path gets this release too. agent owns /opt/spekk and
	// its home, so root does not follow a link that agent can put there.
	install := fmt.Sprintf(`set -e
installed="/opt/spekk/agent-client"
cli="/opt/spekk/cli/spekk"
cli_dir=$(dirname "$cli")
cli_link="/usr/local/bin/spekk"
owner="agent"
group=$(id -gn "$owner")
owner_home=$(getent passwd "$owner" | cut -d: -f6)
agent_upload="$1"
cli_upload="$2"
trap 'rm -f "$agent_upload" "$cli_upload" ${next:+"$next"} ${next_cli:+"$next_cli"}' EXIT
check() { echo "$2  $1" | sha256sum -c --status || { echo "spekk: $(basename "$1") does not match the release sha256" >&2; exit 1; }; }
check "$agent_upload" '%s'
check "$cli_upload" '%s'
if [ -L "$cli_dir" ]; then rm -f "$cli_dir"; fi
mkdir -p "$(dirname "$installed")" "$cli_dir"
next=$(mktemp "$installed.XXXXXX")
install -m 755 "$agent_upload" "$next"
next_cli=$(mktemp "$cli.XXXXXX")
install -m 755 -o "$owner" -g "$group" "$cli_upload" "$next_cli"
mv -f "$next" "$installed"
mv -f "$next_cli" "$cli"
chown -h "$owner:$group" "$cli_dir"
ln -sf "$cli" "$cli_link"
if [ -n "$owner_home" ] && { [ -e "$owner_home/.local/bin/spekk" ] || [ -L "$owner_home/.local/bin/spekk" ]; }; then
  if [ -L "$owner_home/.local" ] || [ -L "$owner_home/.local/bin" ]; then
    echo "spekk: skipped $owner_home/.local/bin/spekk: a directory in that path is a link"
  else
    ln -sfn "$cli" "$owner_home/.local/bin/spekk"
  fi
fi
%s`, agentSHA256, cliSHA256, script)
	// Resolve the upload paths before sudo can change HOME or the working directory.
	return privilegedScript(user, install, fmt.Sprintf(`"$HOME/%s"`, stagedBinary), fmt.Sprintf(`"$HOME/%s"`, stagedCLI))
}

// sudoWrap base64-encodes a script and pipes it through `sudo bash`, running
// it as root without the quoting problems heredocs and nested quotes pose.
// The login user must have passwordless sudo.
//
// shellArgs become the script's positional parameters. They build here, in the
// same expression as the `sudo bash` token, because bash reads the script on
// stdin: anything inserted between that token and `-s --` changes the script
// that runs.
func sudoWrap(script string, shellArgs ...string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	args := ""
	if len(shellArgs) > 0 {
		args = " -s -- " + strings.Join(shellArgs, " ")
	}
	return fmt.Sprintf("echo '%s' | base64 -d | sudo bash%s", encoded, args)
}

// sshUser returns the login user for a sandbox, defaulting to root.
func sshUser(meta *SandboxMeta) string {
	if meta.SSHUser != "" {
		return meta.SSHUser
	}
	return "root"
}

// --- Helpers ---

// KnownHostsFile returns the path to the per-sandbox known_hosts file.
func KnownHostsFile(name string) string {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		dir = config.DefaultDir()
	}
	return filepath.Join(dir, "known_hosts", name)
}

// sshHostKeyOpts returns SSH options for host key verification.
// On first connection (no known_hosts file), it uses accept-new to record the key.
// On subsequent connections, it uses strict verification against the stored key.
func sshHostKeyOpts(name string) []string {
	khFile := KnownHostsFile(name)
	if _, err := os.Stat(khFile); err == nil {
		// Known hosts file exists — verify host key strictly
		return []string{
			"-o", "StrictHostKeyChecking=yes",
			"-o", fmt.Sprintf("UserKnownHostsFile=%s", khFile),
		}
	}
	// First connection — accept and record the key
	os.MkdirAll(filepath.Dir(khFile), 0o700)
	return []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", fmt.Sprintf("UserKnownHostsFile=%s", khFile),
	}
}

// sshArgs returns the base SSH arguments for connecting to a sandbox,
// using the stored key if available.
func sshArgs(sandbox *SandboxMeta, name string) []string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if sandbox.SSHKeyPath != "" {
		args = append(args, "-i", sandbox.SSHKeyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", sshUser(sandbox), sandbox.IP))
	return args
}

// generateSSHKeyPair creates an ed25519 SSH key pair for a sandbox.
// Returns the path to the private key.
func generateSSHKeyPair(name string) (string, error) {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config dir: %w", err)
	}
	keysDir := filepath.Join(dir, "keys")
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		return "", fmt.Errorf("creating keys dir: %w", err)
	}
	keyPath := filepath.Join(keysDir, name)

	// Remove existing key files if any
	os.Remove(keyPath)
	os.Remove(keyPath + ".pub")

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-C", fmt.Sprintf("spekk-%s", name))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	// Restrict permissions on private key
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return "", fmt.Errorf("setting key permissions: %w", err)
	}

	return keyPath, nil
}

func waitForDroplet(client *Client, dropletID int) (string, error) {
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		droplet, err := client.GetDroplet(dropletID)
		if err != nil {
			return "", err
		}
		if droplet.Status == "active" {
			ip := droplet.PublicIP()
			if ip != "" {
				return ip, nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return "", fmt.Errorf("droplet %d did not become active within 5 minutes", dropletID)
}

// waitReady is the seam Create waits through. It is a variable so a test
// can reach the steps after machine creation without a long wait.
var waitReady = waitForProvisioning

// DefaultProvisionTimeout is how long Create waits for cloud-init when the
// operator sets no --provision-timeout. Ten minutes was too short: an apt
// upgrade plus the nodesource repository has taken eighteen on a slow night,
// and the droplet then finished provisioning after Create had given up on it.
const DefaultProvisionTimeout = 30 * time.Minute

// ParseProvisionTimeout reads the --provision-timeout flag. An empty value
// is the default; anything else must be a positive Go duration such as 45m.
func ParseProvisionTimeout(s string) (time.Duration, error) {
	if strings.TrimSpace(s) == "" {
		return DefaultProvisionTimeout, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --provision-timeout %q: use a duration such as 30m or 1h", s)
	}
	if d <= 0 {
		return 0, fmt.Errorf("invalid --provision-timeout %q: must be longer than zero", s)
	}
	return d, nil
}

// provisionPollInterval is how often the wait asks the machine whether the
// marker is there. provisionReportInterval is how often it tells the
// operator what it saw, so a long wait reads as alive rather than hung.
// Both are variables so a test can run the loop without the real delays.
var (
	provisionPollInterval   = 5 * time.Second
	provisionReportInterval = time.Minute
)

// cloudInitLog is where cloud-init writes the output of every runcmd step,
// so its last line says what provisioning is doing right now.
const cloudInitLog = "/var/log/cloud-init-output.log"

// provisionProbeScript is one SSH round trip that reports everything the
// wait wants to know: what cloud-init says about itself, the marker, and the
// last line of its log. The status is read before the marker is tested: the
// marker is the last runcmd step, so a status of "done" read first cannot
// miss a marker that the same moment wrote (#282). The log line goes through
// printf, quoted, so a line with a glob character or a run of spaces arrives
// as it was written. Each line of the status gets a "cloud-init: " prefix.
const provisionProbeScript = `long=$(cloud-init status --long 2>/dev/null)
marker=no; test -f ` + provisionedMarker + ` && marker=yes
log=$(tail -n 1 ` + cloudInitLog + ` 2>/dev/null)
printf 'marker=%s\nlog=%s\n' "$marker" "$log"
printf '%s\n' "$long" | sed 's/^/cloud-init: /'`

// provisionProbe is what one poll of a provisioning machine reports.
type provisionProbe struct {
	// marker is true once /opt/spekk/.provisioned exists.
	marker bool
	// cloudInit is what `cloud-init status` reports: running, done, error,
	// or "" when the machine could not be asked.
	cloudInit string
	// hasErrors is true when the errors of `cloud-init status --long` are not
	// empty.
	hasErrors bool
	// status is the full output of `cloud-init status --long`, for the
	// error message.
	status string
	// lastLog is the last line of cloud-init's output log.
	lastLog string
}

// parseProvisionProbe reads the output of provisionProbeScript. Output that
// is not in that form, including the empty string a failed connection
// returns, reads as "nothing known yet".
func parseProvisionProbe(out string) provisionProbe {
	var p provisionProbe
	var status []string
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, "cloud-init: "); ok {
			status = append(status, rest)
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "marker":
			p.marker = value == "yes"
		case "log":
			p.lastLog = strings.TrimSpace(value)
		}
	}
	p.status = strings.TrimSpace(strings.Join(status, "\n"))
	p.cloudInit, p.hasErrors = readCloudInitStatus(status)
	return p
}

// readCloudInitStatus reads the state and whether there are errors from the
// lines of `cloud-init status --long`. The errors are a YAML list: "[]" when
// there are none, else inline ("errors: [...]") or one "- " item on each
// line after "errors:".
func readCloudInitStatus(lines []string) (state string, hasErrors bool) {
	inErrors := false
	for _, line := range lines {
		if inErrors {
			if item := strings.TrimSpace(line); strings.HasPrefix(item, "- ") {
				hasErrors = true
				continue
			}
			inErrors = false
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch key {
		case "status":
			state = value
		case "errors":
			if value == "" {
				inErrors = true
			} else if value != "[]" {
				hasErrors = true
			}
		}
	}
	return state, hasErrors
}

// probeProvisioning is the seam the wait polls through, so a test can drive
// the loop without a machine.
var probeProvisioning = func(ip, keyPath, name, user string) provisionProbe {
	return parseProvisionProbe(runSSH(ip, keyPath, name, user, provisionProbeScript))
}

// markerGrace is how long the wait polls for the marker after cloud-init
// reports "done" without it. The marker is the last runcmd step, and on a
// real machine it has arrived in the same second as the "finished" line
// (#282). A variable so a test can make it short.
var markerGrace = 30 * time.Second

// cloudInitFailed reports a real cloud-init failure, or nil. Only this stops
// the wait early: "done" without the marker gets the grace period.
func cloudInitFailed(p provisionProbe) error {
	if p.cloudInit != "error" && !p.hasErrors {
		return nil
	}
	return fmt.Errorf("cloud-init reported an error before writing %s\n%s", provisionedMarker, statusOrPlaceholder(p))
}

// statusOrPlaceholder is the cloud-init status output for an error message.
func statusOrPlaceholder(p provisionProbe) string {
	if p.status == "" {
		return "(no cloud-init status)"
	}
	return p.status
}

// lastLogOrPlaceholder is the last cloud-init log line as a progress note.
func lastLogOrPlaceholder(p provisionProbe) string {
	if p.lastLog == "" {
		return "(no cloud-init output yet)"
	}
	return p.lastLog
}

// waitForProvisioning logs in as user, which is the user the provider
// recorded: root on a droplet, and a sudo user on a cloud whose image does
// not admit root.
func waitForProvisioning(ip, keyPath, name, user string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultProvisionTimeout
	}
	deadline := time.Now().Add(timeout)

	// Wait for SSH connectivity
	fmt.Fprintln(os.Stderr, "Waiting for SSH connectivity...")
	for time.Now().Before(deadline) {
		if sshPortOpen(ip) {
			break
		}
		time.Sleep(provisionPollInterval)
	}

	fmt.Fprintf(os.Stderr, "Waiting up to %s for cloud-init provisioning to complete...\n", timeout)
	return waitForMarker(deadline, func() provisionProbe { return probeProvisioning(ip, keyPath, name, user) })
}

// sshPortOpen is the seam the wait checks port 22 through, so a test can run
// the real wait against a fake probe.
var sshPortOpen = func(ip string) bool { return checkTCPPort(ip, 22) }

// waitForMarker polls probe until the marker appears, cloud-init reports a
// failure, or the time is up. The time is the deadline until cloud-init
// reports "done", and from then on the grace period, also past the deadline.
// At most once a report interval it prints how long it has waited and what
// cloud-init last logged.
func waitForMarker(deadline time.Time, probe func() provisionProbe) error {
	start := time.Now()
	lastReport := start
	var last provisionProbe
	var doneAt time.Time
	for {
		if doneAt.IsZero() && !time.Now().Before(deadline) {
			break
		}
		if !doneAt.IsZero() && !time.Now().Before(doneAt.Add(markerGrace)) {
			return fmt.Errorf("cloud-init finished without writing %s within %s\n%s\nLast log line: %s",
				provisionedMarker, markerGrace, statusOrPlaceholder(last), lastLogOrPlaceholder(last))
		}
		last = probe()
		if last.marker {
			return nil
		}
		if err := cloudInitFailed(last); err != nil {
			return fmt.Errorf("%w\nLast log line: %s", err, lastLogOrPlaceholder(last))
		}
		if last.cloudInit == "done" && doneAt.IsZero() {
			doneAt = time.Now()
			fmt.Fprintf(os.Stderr, "cloud-init is done; waiting up to %s for %s...\n", markerGrace, provisionedMarker)
		}
		if time.Since(lastReport) >= provisionReportInterval {
			lastReport = time.Now()
			fmt.Fprintf(os.Stderr, "Still provisioning after %s. cloud-init: %s\n",
				time.Since(start).Round(time.Second), lastLogOrPlaceholder(last))
		}
		time.Sleep(provisionPollInterval)
	}
	return fmt.Errorf("provisioning did not complete within %s; cloud-init may still be running (last log line: %s)",
		time.Since(start).Round(time.Second), lastLogOrPlaceholder(last))
}

func checkTCPPort(ip string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// runSSH runs command on the host as user and returns trimmed stdout, or ""
// on error.
func runSSH(ip, keyPath, name, user, command string) string {
	out, err := runSSHStdout(ip, keyPath, name, user, command)
	if err != nil {
		return ""
	}
	return out
}

// runSSHStdout runs command on the host as user and returns its stdout only,
// so an ssh warning cannot become part of the output. A failure carries
// what ssh or the command wrote to stderr.
func runSSHStdout(ip, keyPath, name, user, command string) (string, error) {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", user, ip), command)
	out, err := exec.Command("ssh", args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
				return "", fmt.Errorf("%w: %s", err, stderr)
			}
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runSSHCombined runs command on the host as user, returning combined output
// and any error.
func runSSHCombined(ip, keyPath, name, user, command string) (string, error) {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", user, ip), command)
	out, err := sshExec(args, "")
	return string(out), err
}

// runSecretScript runs a script that holds a secret as root on the host. The
// script goes on the stdin of ssh, never in its argument list: each process
// on this host can read the argument list of ssh in /proc/<pid>/cmdline, and
// a secret in base64 there is still the secret. The remote command is only
// "bash -s", or "sudo bash -s" for a login user that is not root.
func runSecretScript(ip, keyPath, name, user, script string) (string, error) {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	shell := "bash -s"
	if user != "root" {
		shell = "sudo bash -s"
	}
	args = append(args, fmt.Sprintf("%s@%s", user, ip), shell)
	// bash reads a script on stdin one command at a time, so a command in
	// it that reads stdin would eat the rest of the script. The braces make
	// bash read the whole script first, and give each command /dev/null.
	out, err := sshExec(args, "{\n"+script+"\n} </dev/null\n")
	return string(out), err
}

// sshBatchArgs builds the ssh arguments for a non-interactive command on a
// sandbox: host-key options, a connect timeout, no prompting, the sandbox key
// if there is one, and the login user.
func sshBatchArgs(sandbox *SandboxMeta, name string) []string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10", "-o", "BatchMode=yes")
	if sandbox.SSHKeyPath != "" {
		args = append(args, "-i", sandbox.SSHKeyPath)
	}
	return append(args, fmt.Sprintf("%s@%s", sshUser(sandbox), sandbox.IP))
}

func sshCheck(sandbox *SandboxMeta, name, command string) string {
	cmd := exec.Command("ssh", append(sshBatchArgs(sandbox, name), command)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// injectCredentials writes the env file from the credentials in the
// environment of the CLI, and in the same command the sandbox marker, which
// holds no secret.
func injectCredentials(ip, keyPath, name, user, agentToken string, mode AuthMode, marker []byte) error {
	envVars := map[string]string{
		"AWS_ACCESS_KEY_ID":     os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY": os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"AWS_DEFAULT_REGION":    os.Getenv("AWS_DEFAULT_REGION"),
		"GITHUB_TOKEN":          os.Getenv("GITHUB_TOKEN"),
		oauthTokenVar:           os.Getenv(oauthTokenVar),
	}
	spekkHost := os.Getenv("SPEKK_HOST")

	if err := writeEnvFile(ip, keyPath, name, user, buildEnvContent(mode, envVars, name, agentToken, spekkHost), marker); err != nil {
		return err
	}
	if mode == AuthSubscription && os.Getenv(codexAuthFileVar) != "" {
		return injectCodexAuth(ip, keyPath, name, user, os.Getenv(codexAuthFileVar))
	}
	return nil
}

// writeEnvFile writes envContent to /etc/spekk/agent.env, and the marker.
func writeEnvFile(ip, keyPath, name, user, envContent string, marker []byte) error {
	out, err := runSecretScript(ip, keyPath, name, user, buildInjectScript(envContent, marker))
	if err != nil {
		return fmt.Errorf("SSH command failed: %s\n%s", err, out)
	}
	if strings.Contains(out, markerNotWritten) {
		cause := strings.TrimSpace(strings.Replace(out, markerNotWritten, "", 1))
		fmt.Fprintf(os.Stderr, "Warning: could not write %s. The sandbox works without it.\n%s\n", sandboxMarker, cause)
	}
	return nil
}

// buildEnvContent constructs the env file content for credential injection.
// The model credential comes from the auth mode; everything after it is the
// same whatever the sandbox pays with. Exported for testing.
func buildEnvContent(mode AuthMode, envVars map[string]string, name, agentToken, spekkHost string) string {
	envLines := append(authLines(mode, envVars),
		"GITHUB_TOKEN="+envVars["GITHUB_TOKEN"],
		"SPEKK_HOST="+bareHost(spekkHost),
		"SPEKK_AGENT_TOKEN="+agentToken,
		"WORKSPACE=/opt/spekk/workspace",
		"SPEKK_AGENT_NAME=spekk-"+name,
	)
	return strings.Join(envLines, "\n") + "\n"
}

// bareHost is SPEKK_HOST with no scheme and no trailing slash, the form the
// agent client reads.
func bareHost(spekkHost string) string {
	return strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(spekkHost, "https://"), "http://"), "/")
}

// markerNotWritten is the line the inject script prints when it cannot
// write the sandbox marker, so injectCredentials can warn about it.
const markerNotWritten = "spekk: marker not written"

// buildInjectScript produces the script for injecting credentials
// and writing the sandbox marker. It base64-encodes both to prevent shell
// injection. The marker goes to a temp file and then mv -f, so the write does
// not follow a symlink at the marker path. A failed marker write prints
// markerNotWritten and does not fail the command: the marker is information
// only. Exported for testing.
func buildInjectScript(envContent string, marker []byte) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(envContent))
	encodedMarker := base64.StdEncoding.EncodeToString(marker)
	return fmt.Sprintf(
		`echo '%s' | base64 -d > /etc/spekk/agent.env && chown agent:agent /etc/spekk/agent.env && chmod 600 /etc/spekk/agent.env && `+
			`{ m=$(mktemp %s.XXXXXX) && echo '%s' | base64 -d > "$m" && chmod 644 "$m" && mv -f "$m" %s || { [ -z "$m" ] || rm -f "$m"; echo '%s'; }; }`,
		encoded, sandboxMarker, encodedMarker, sandboxMarker, markerNotWritten,
	)
}

func configureGitCredentials(ip, keyPath, name, user string) error {
	ghToken := os.Getenv("GITHUB_TOKEN")

	if out, err := runSecretScript(ip, keyPath, name, user, buildGitCredentialScript(ghToken)); err != nil {
		return fmt.Errorf("SSH command failed: %s\n%s", err, out)
	}
	return nil
}

// buildGitCredentialScript produces the script for configuring git credentials.
// It base64-encodes the token to prevent shell injection. Exported for testing.
func buildGitCredentialScript(ghToken string) string {
	encodedToken := base64.StdEncoding.EncodeToString([]byte(ghToken))
	return fmt.Sprintf(`set -e
TOKEN=$(echo '%s' | base64 -d)
su - agent -c 'git config --global credential.helper store'
su - agent -c "cat > ~/.git-credentials" <<< "https://x-access-token:${TOKEN}@github.com"
su - agent -c 'chmod 600 ~/.git-credentials'
echo "${TOKEN}" | su - agent -c 'gh auth login --with-token'
su - agent -c 'gh auth status' >/dev/null 2>&1 || { echo 'spekk: gh auth status fails: the agent user is not logged in to GitHub' >&2; exit 1; }`,
		encodedToken,
	)
}

func resolveProject(client *Client, projectValue string) (string, string, error) {
	// If it looks like a UUID, use it directly
	if len(projectValue) == 36 && strings.Count(projectValue, "-") == 4 {
		return projectValue, projectValue, nil
	}

	projects, err := client.ListProjects()
	if err != nil {
		return "", "", fmt.Errorf("listing projects: %w", err)
	}
	for _, p := range projects {
		if p.Name == projectValue {
			return p.ID, p.Name, nil
		}
	}

	var names []string
	for _, p := range projects {
		names = append(names, "  - "+p.Name)
	}
	return "", "", fmt.Errorf("no project found with name %q\nAvailable projects:\n%s", projectValue, strings.Join(names, "\n"))
}

func generateAgentToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func padRow(name, kind, ip, region, status, created, expires string) string {
	return fmt.Sprintf("%-20s  %-10s  %-18s  %-14s  %-12s  %-20s  %s", name, kind, ip, region, status, created, expires)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
