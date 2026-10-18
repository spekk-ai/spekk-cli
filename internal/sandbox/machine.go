package sandbox

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/version"
)

// Machine mode is the interface of `spekk sandbox create --machine` and
// `spekk sandbox destroy --machine` for a server worker: the secrets come on
// stdin, and the progress goes to stdout as one JSON line for each stage.
// See specs/sandbox-machine-mode/.

// The stages of a machine-mode create, in order, and the line for a failure.
const (
	stageCreating     = "creating"
	stageProvisioning = "provisioning"
	stageRegistering  = "registering"
	stageOnline       = "online"
	stageFailed       = "failed"
)

// maxMachineInput is the most stdin that machine mode reads. The real input
// is well under 4 KiB.
const maxMachineInput = 64 << 10

// machineInput is the stdin of a machine-mode create, after it is checked.
type machineInput struct {
	doToken           string
	registrationToken string
	controlHost       string // base URL: scheme and host
	githubToken       string
	repository        string // owner/name
	auth              AuthMode
	modelEnv          map[string]string // the variables of the model credential
}

// secrets lists each secret of the input, for redaction.
func (in *machineInput) secrets() []string {
	s := []string{in.doToken, in.registrationToken, in.githubToken}
	for _, v := range in.modelEnv {
		s = append(s, v)
	}
	return s
}

// repositoryRe is owner/name as GitHub allows them. It also keeps the value
// safe to quote in a shell script.
var repositoryRe = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// decodeStrict decodes exactly one JSON object from data into v. A field
// that v does not have, or data after the object, is an error.
func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&json.RawMessage{}); err != io.EOF {
		return errors.New("data after the JSON object")
	}
	return nil
}

// readStdinJSON reads r to its end, at most maxMachineInput, and decodes it
// strictly into v.
func readStdinJSON(r io.Reader, v any) error {
	data, err := io.ReadAll(io.LimitReader(r, maxMachineInput+1))
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) > maxMachineInput {
		return fmt.Errorf("stdin is larger than %d bytes", maxMachineInput)
	}
	if err := decodeStrict(data, v); err != nil {
		return fmt.Errorf("stdin is not the expected JSON object: %w", err)
	}
	return nil
}

// requireFields returns an error that names each field whose value is empty.
// fields alternates name and value.
func requireFields(fields ...string) error {
	var missing []string
	for i := 0; i+1 < len(fields); i += 2 {
		if strings.TrimSpace(fields[i+1]) == "" {
			missing = append(missing, fields[i])
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("stdin is missing a value for %s", strings.Join(missing, ", "))
	}
	return nil
}

// readMachineInput reads and checks the stdin of a machine-mode create.
func readMachineInput(r io.Reader) (*machineInput, error) {
	var raw struct {
		DigitalOceanToken string          `json:"digitalocean_token"`
		RegistrationToken string          `json:"registration_token"`
		SpekkHost         string          `json:"spekk_host"`
		GitHubToken       string          `json:"github_token"`
		GitHubRepository  string          `json:"github_repository"`
		Harness           string          `json:"harness"`
		ModelCredential   json.RawMessage `json:"model_credential"`
	}
	if err := readStdinJSON(r, &raw); err != nil {
		return nil, err
	}
	if err := requireFields(
		"digitalocean_token", raw.DigitalOceanToken,
		"registration_token", raw.RegistrationToken,
		"spekk_host", raw.SpekkHost,
		"github_token", raw.GitHubToken,
		"github_repository", raw.GitHubRepository,
		"harness", raw.Harness,
		"model_credential", string(raw.ModelCredential),
	); err != nil {
		return nil, err
	}
	if raw.Harness != "claude-code" {
		return nil, fmt.Errorf("harness %q is not claude-code, the only harness that machine mode installs", raw.Harness)
	}
	owner, repo, _ := strings.Cut(raw.GitHubRepository, "/")
	if !repositoryRe.MatchString(raw.GitHubRepository) || owner == "." || owner == ".." || repo == "." || repo == ".." {
		return nil, fmt.Errorf("github_repository %q is not <owner>/<repository>", raw.GitHubRepository)
	}
	base, err := controlHostBase(raw.SpekkHost)
	if err != nil {
		return nil, fmt.Errorf("spekk_host: %w", err)
	}
	in := &machineInput{
		doToken:           raw.DigitalOceanToken,
		registrationToken: raw.RegistrationToken,
		controlHost:       base,
		githubToken:       raw.GitHubToken,
		repository:        raw.GitHubRepository,
	}
	if in.auth, in.modelEnv, err = readModelCredential(raw.ModelCredential); err != nil {
		return nil, err
	}
	return in, nil
}

// readModelCredential reads model_credential. Each kind has its own fields,
// so a field of another kind is an unknown field.
func readModelCredential(data []byte) (AuthMode, map[string]string, error) {
	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return "", nil, fmt.Errorf("model_credential is not a JSON object: %w", err)
	}
	switch head.Kind {
	case "none":
		var c struct {
			Kind string `json:"kind"`
		}
		if err := decodeStrict(data, &c); err != nil {
			return "", nil, fmt.Errorf("model_credential: %w", err)
		}
		return AuthNone, map[string]string{}, nil
	case "anthropic_api_key":
		var c struct {
			Kind   string `json:"kind"`
			APIKey string `json:"api_key"`
		}
		if err := decodeStrict(data, &c); err != nil {
			return "", nil, fmt.Errorf("model_credential: %w", err)
		}
		if err := requireFields("model_credential.api_key", c.APIKey); err != nil {
			return "", nil, err
		}
		return AuthAPIKey, map[string]string{apiKeyVar: c.APIKey}, nil
	case "bedrock":
		var c struct {
			Kind               string `json:"kind"`
			AWSAccessKeyID     string `json:"aws_access_key_id"`
			AWSSecretAccessKey string `json:"aws_secret_access_key"`
			AWSRegion          string `json:"aws_region"`
		}
		if err := decodeStrict(data, &c); err != nil {
			return "", nil, fmt.Errorf("model_credential: %w", err)
		}
		if err := requireFields(
			"model_credential.aws_access_key_id", c.AWSAccessKeyID,
			"model_credential.aws_secret_access_key", c.AWSSecretAccessKey,
			"model_credential.aws_region", c.AWSRegion,
		); err != nil {
			return "", nil, err
		}
		return AuthBedrock, map[string]string{
			"AWS_ACCESS_KEY_ID":     c.AWSAccessKeyID,
			"AWS_SECRET_ACCESS_KEY": c.AWSSecretAccessKey,
			"AWS_DEFAULT_REGION":    c.AWSRegion,
		}, nil
	}
	return "", nil, fmt.Errorf("model_credential.kind %q is not none, anthropic_api_key, or bedrock", head.Kind)
}

// stageLine is one line of machine-mode stdout.
type stageLine struct {
	Stage     string `json:"stage"`
	DropletID int    `json:"droplet_id,omitempty"`
	IP        string `json:"ip,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	At        string `json:"at,omitempty"`
	Message   string `json:"message,omitempty"`
}

// stageKeepalive is how often a machine-mode create writes the last stage
// line again while it waits. The worker stops a create that writes no line
// for 20 minutes, and the cloud-init wait alone can take 30. A var, so a
// test can make it short.
var stageKeepalive = 30 * time.Second

// stageWriter writes the stage lines of one machine-mode command, and
// remembers the last stage that started, for the failed line.
type stageWriter struct {
	out     io.Writer
	stage   string
	secrets []string

	mu   sync.Mutex
	last stageLine     // the line that the keepalive writes again
	stop chan struct{} // stops the keepalive; nil until the first line
	done bool          // set by finish: no line comes after the final line
}

// emit writes the line of a stage that starts. The first line starts the
// keepalive.
func (s *stageWriter) emit(line stageLine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stage = line.Stage
	s.last = line
	s.write(line)
	if s.stop == nil {
		s.stop = make(chan struct{})
		go s.keepalive(s.stop)
	}
}

// keepalive writes the last stage line again, unchanged, on each tick until
// stop closes. The worker sets the same stage again, so the line needs no
// field of its own.
func (s *stageWriter) keepalive(stop <-chan struct{}) {
	tick := time.NewTicker(stageKeepalive)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			s.mu.Lock()
			if !s.done {
				s.write(s.last)
			}
			s.mu.Unlock()
		}
	}
}

// finish writes the online or failed line. It stops the keepalive under the
// lock first, so the final line is the last line.
func (s *stageWriter) finish(line stageLine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != nil {
		close(s.stop)
	}
	s.done = true
	s.write(line)
}

// write writes line as one JSON line. A message keeps < and > as they are.
func (s *stageWriter) write(line stageLine) {
	enc := json.NewEncoder(s.out)
	enc.SetEscapeHTML(false)
	enc.Encode(line)
}

// fail writes the failed line for err, and returns err with each secret
// taken out, for the error that the command prints on stderr.
func (s *stageWriter) fail(err error) error {
	msg := redact(err.Error(), s.secrets)
	s.finish(stageLine{Stage: stageFailed, At: s.stage, Message: msg})
	return errors.New(msg)
}

// redact replaces each secret in text with [redacted]. No code path is
// known to put a secret in an error, so this is a second guard.
func redact(text string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[redacted]")
		}
	}
	return text
}

// WriteMachineFailure writes the failed line for an error before a
// machine-mode command reads its input, such as a bad flag.
func WriteMachineFailure(out io.Writer, err error) {
	(&stageWriter{out: out, stage: stageCreating}).fail(err)
}

// labelPartRe is a label key or value that is valid on each provider.
var labelPartRe = regexp.MustCompile(`^[a-z0-9_-]{1,63}$`)

// ParseLabels reads the --label key=value flags. A label that spekk sets
// itself is refused, so a caller cannot change the kind of a sandbox.
func ParseLabels(flags []string) (map[string]string, error) {
	labels := map[string]string{}
	for _, f := range flags {
		k, v, ok := strings.Cut(f, "=")
		if !ok || !labelPartRe.MatchString(k) || !labelPartRe.MatchString(v) {
			return nil, fmt.Errorf("invalid --label %q: use key=value, each lowercase [a-z0-9_-] and 1 to 63 characters", f)
		}
		switch k {
		case labelKind, labelExpires, labelHost, "spekk-sandbox":
			return nil, fmt.Errorf("invalid --label %q: spekk sets %s itself", f, k)
		}
		labels[k] = v
	}
	return labels, nil
}

// releaseVersionRe is the version of a CLI built from a stable release tag.
var releaseVersionRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// machineRelease is the release that a machine-mode create deploys the
// agent from: the pinned one, or the release of this CLI. The agent of
// another release can lack the git credential helper that machine mode
// configures, so a CLI that is not a stable release must pin one.
func machineRelease(pinned string) (string, error) {
	if pinned != "" {
		return pinned, nil
	}
	if releaseVersionRe.MatchString(version.Version) {
		return version.Version, nil
	}
	return "", fmt.Errorf("this spekk is version %q, not a stable release, so machine mode needs --release <tag> to choose the agent it deploys", version.Version)
}

// machineDOClient is the seam machine mode builds its DigitalOcean client
// through, so a test can point it at a fake API.
var machineDOClient = func(token string) *Client {
	return NewClientWithToken(token, baseURL)
}

// CreateMachine is `spekk sandbox create --machine`. It reads the input on
// stdin, and writes the stage lines to stdout. On a failure, it keeps the
// record of a droplet that exists and revokes nothing: the caller destroys
// the droplet with DestroyMachine and revokes the agent.
func CreateMachine(stdin io.Reader, stdout io.Writer, opts CreateOptions) error {
	st := &stageWriter{out: stdout, stage: stageCreating}
	in, err := readMachineInput(stdin)
	if err != nil {
		return st.fail(err)
	}
	st.secrets = in.secrets()
	release, err := machineRelease(opts.Release)
	if err != nil {
		return st.fail(err)
	}
	if existing, err := GetSandbox(opts.Name); err != nil {
		return st.fail(err)
	} else if existing != nil {
		return st.fail(fmt.Errorf("sandbox %q already exists (%s)", opts.Name, machineRef(existing)))
	}
	st.emit(stageLine{Stage: stageCreating})

	fmt.Fprintln(os.Stderr, "Fetching sandbox release artifacts...")
	artifacts, err := fetchArtifacts(release)
	if err != nil {
		return st.fail(fmt.Errorf("fetching release artifacts: %w", err))
	}
	defer func() { artifacts.cleanup() }()
	fmt.Fprintf(os.Stderr, "Using sandbox release %s\n", artifacts.Version)
	opts.CloudInit = artifacts.CloudInit

	p := &DOProvider{client: machineDOClient(in.doToken)}
	meta := &SandboxMeta{
		Provider:  p.Name(),
		Auth:      string(in.auth),
		Status:    "provisioning",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Kind:      opts.Kind,
		CreatedBy: hostname(),
	}
	if meta.Kind == "" {
		meta.Kind = KindPersistent
	}
	if !opts.ExpiresAt.IsZero() {
		meta.ExpiresAt = opts.ExpiresAt.UTC().Format(time.RFC3339)
	}
	createErr := p.Create(opts.Name, opts, meta)
	if namesMachine(meta) {
		if err := SaveSandbox(opts.Name, meta); err != nil {
			return st.fail(errors.Join(createErr, fmt.Errorf("saving metadata for %s: %w", machineRef(meta), err)))
		}
	}
	if createErr != nil {
		return st.fail(createErr)
	}
	labelMachine(p, meta)
	st.emit(stageLine{Stage: stageProvisioning, DropletID: meta.DropletID, IP: meta.IP})

	if err := waitReady(meta.IP, meta.SSHKeyPath, opts.Name, sshUser(meta), opts.ProvisionTimeout); err != nil {
		return st.fail(fmt.Errorf("waiting for provisioning: %w", err))
	}
	st.emit(stageLine{Stage: stageRegistering})

	agentToken := generateAgentToken()
	fmt.Fprintln(os.Stderr, "Registering the agent with the registration token...")
	agentID, err := newRegistrationHost(in.controlHost, in.registrationToken).registerWithToken(opts.Name, agentToken)
	if err != nil {
		return st.fail(fmt.Errorf("registering the agent: %w", err))
	}
	fmt.Fprintf(os.Stderr, "Registered agent %s.\n", agentID)
	meta.AgentID = agentID
	if err := SaveSandbox(opts.Name, meta); err != nil {
		return st.fail(fmt.Errorf("saving metadata: %w", err))
	}

	if err := equipMachine(meta, opts.Name, agentToken, in, artifacts); err != nil {
		return st.fail(err)
	}

	meta.Status = "active"
	if err := SaveSandbox(opts.Name, meta); err != nil {
		return st.fail(fmt.Errorf("saving metadata: %w", err))
	}
	st.finish(stageLine{Stage: stageOnline, AgentID: agentID})
	return nil
}

// equipMachine is the seam CreateMachine equips through, so a test can
// reach the steps around it without an SSH connection.
var equipMachine = equipMachineSandbox

// equipMachineSandbox runs the steps of equipSandbox for machine mode: the
// env file comes from the input, and git reaches the one repository of the
// input through the credential helper, not through a stored token.
func equipMachineSandbox(meta *SandboxMeta, name, agentToken string, in *machineInput, artifacts *releaseArtifacts) *stageError {
	user := sshUser(meta)

	if err := fetchAgent(meta, name, artifacts); err != nil {
		return &stageError{"fetching agent binary", err}
	}

	fmt.Fprintln(os.Stderr, "Injecting credentials...")
	if err := writeEnvFile(meta.IP, meta.SSHKeyPath, name, user, buildMachineEnvContent(in, name, agentToken, meta.AgentID), markerContent(meta)); err != nil {
		return &stageError{"injecting credentials", err}
	}

	fmt.Fprintf(os.Stderr, "Cloning %s...\n", in.repository)
	if out, err := runSecretScript(meta.IP, meta.SSHKeyPath, name, user, buildCloneScript(in.githubToken, in.repository)); err != nil {
		return &stageError{"cloning " + in.repository, fmt.Errorf("%w\n%s", err, strings.TrimSpace(out))}
	}

	fmt.Fprintln(os.Stderr, "Deploying agent binary...")
	if err := deployAgent(meta.IP, meta.SSHKeyPath, name, user, artifacts); err != nil {
		return &stageError{"deploying agent", err}
	}
	return nil
}

// buildMachineEnvContent is the env file of a machine-mode sandbox. It has
// no GITHUB_TOKEN: the installation token lives 1 hour, and git gets a new
// one through the credential helper, which reads SPEKK_AGENT_ID.
func buildMachineEnvContent(in *machineInput, name, agentToken, agentID string) string {
	lines := append(authLines(in.auth, in.modelEnv),
		"SPEKK_HOST="+bareHost(in.controlHost),
		"SPEKK_AGENT_TOKEN="+agentToken,
		"SPEKK_AGENT_ID="+agentID,
		"WORKSPACE=/opt/spekk/workspace",
		"SPEKK_AGENT_NAME=spekk-"+name,
	)
	return strings.Join(lines, "\n") + "\n"
}

// gitCredentialHelper is the git credential helper of a machine-mode
// sandbox: the agent binary asks the control host for an installation token.
const gitCredentialHelper = "/opt/spekk/agent-client git-credential"

// buildCloneScript clones the repository into the workspace as agent, then
// configures the credential helper. The token reaches git in an environment
// variable of the git process, which only agent and root can read. The
// helper of the clone is a shell function that git gets in an argument, and
// the argument holds the name of the variable, not the token. The remote URL
// has no token. The global helper is set after the clone, so the clone does
// not call it.
func buildCloneScript(token, repository string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(token))
	return fmt.Sprintf(`set -e
SPEKK_CLONE_TOKEN=$(echo '%s' | base64 -d)
export SPEKK_CLONE_TOKEN
install -d -o agent -g agent /opt/spekk/workspace
runuser -u agent -- git -c credential.helper= -c 'credential.helper=!f() { test "$1" = get || return 0; echo username=x-access-token; echo "password=$SPEKK_CLONE_TOKEN"; }; f' clone --quiet 'https://github.com/%s.git' /opt/spekk/workspace
unset SPEKK_CLONE_TOKEN
runuser -u agent -- git config --global credential.https://github.com.helper '%s'
runuser -u agent -- git config --global credential.https://github.com.useHttpPath false`,
		encoded, repository, gitCredentialHelper)
}

// DestroyMachine is `spekk sandbox destroy --machine --force`. It reads the
// DigitalOcean token on stdin, and does not revoke the agent: the caller
// does, and machine mode has no API token.
func DestroyMachine(stdin io.Reader, name string) error {
	var raw struct {
		DigitalOceanToken string `json:"digitalocean_token"`
	}
	if err := readStdinJSON(stdin, &raw); err != nil {
		return err
	}
	if err := requireFields("digitalocean_token", raw.DigitalOceanToken); err != nil {
		return err
	}
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}
	if providerName(sandbox) != "digitalocean" {
		return fmt.Errorf("sandbox %q is on %s; machine mode destroys DigitalOcean sandboxes only", name, providerName(sandbox))
	}
	if err := Destroy(&DOProvider{client: machineDOClient(raw.DigitalOceanToken)}, name, DestroyOptions{Force: true, CallerRevokes: true}); err != nil {
		return errors.New(redact(err.Error(), []string{raw.DigitalOceanToken}))
	}
	return nil
}
