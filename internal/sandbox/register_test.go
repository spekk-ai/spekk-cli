package sandbox

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// stubEquip replaces the SSH steps that equip a machine, and returns the
// agent token each call received.
func stubEquip(t *testing.T, err error) *[]string {
	t.Helper()
	var tokens []string
	orig := equip
	equip = func(meta *SandboxMeta, name, agentToken string, mode AuthMode, artifacts *releaseArtifacts) *stageError {
		tokens = append(tokens, agentToken)
		if err != nil {
			return &stageError{"deploying agent", err}
		}
		return nil
	}
	t.Cleanup(func() { equip = orig })
	return &tokens
}

// createExisting runs create for a machine the operator already has, so no
// cloud is involved and the control host calls are the only network.
func createExisting(t *testing.T, name string) error {
	t.Helper()
	key := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(key, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := checkReady
	checkReady = func(meta *SandboxMeta, name string) error { return nil }
	t.Cleanup(func() { checkReady = orig })
	return Create(nil, CreateOptions{Name: name, IP: "1.2.3.4", SSHKey: key, Auth: AuthBedrock})
}

func TestCreateRegistersBeforeItProvisions(t *testing.T) {
	isolateConfig(t)
	store := useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)

	// The provider is where billing starts, so registration must be done
	// before it is asked for a machine.
	var callsAtProvider []string
	p := &recordingProvider{ip: "9.9.9.9", dropletID: 7}
	origWait := waitReady
	waitReady = func(ip, keyPath, name, user string, timeout time.Duration) error { return nil }
	t.Cleanup(func() { waitReady = origWait })
	tokens := stubEquip(t, nil)
	stderr := captureStderr(t)
	err := Create(providerFunc{p, func() { callsAtProvider = host.calls() }}, CreateOptions{Name: "box", Auth: AuthBedrock})
	out := stderr()
	if err != nil {
		t.Fatalf("create: %v\n%s", err, out)
	}

	if !slices.Equal(callsAtProvider, []string{"POST /api/agents/"}) {
		t.Errorf("calls before the provider = %v, want only the registration", callsAtProvider)
	}
	got, _ := GetSandbox("box")
	if got == nil || got.AgentID == "" || got.Status != "active" {
		t.Fatalf("record = %+v, want an active record with the agent id", got)
	}
	agent := host.agent(got.AgentID)
	if agent == nil {
		t.Fatalf("the record names agent %q, which the control host did not make", got.AgentID)
	}
	if !slices.Equal(*tokens, []string{agent.Token}) {
		t.Errorf("the machine got token %v, want the one the control host made", *tokens)
	}
	if !strings.HasPrefix(agent.SandboxID, "spekk-box-") || len(agent.SandboxID) != len("spekk-box-")+8 {
		t.Errorf("sandbox_id = %q, want spekk-box-<8 hex>", agent.SandboxID)
	}

	// The record holds the id and no credential.
	data, _ := os.ReadFile(store)
	if !strings.Contains(string(data), `"agentId": "`+got.AgentID+`"`) {
		t.Errorf("sandboxes.json has no agentId:\n%s", data)
	}
	for _, secret := range []string{agent.Token, testAPIToken} {
		if strings.Contains(string(data), secret) {
			t.Errorf("sandboxes.json holds a credential %q", secret)
		}
		if strings.Contains(out, secret) {
			t.Errorf("create printed a credential %q:\n%s", secret, out)
		}
	}
}

// providerFunc runs hook when create asks the provider for a machine.
type providerFunc struct {
	*recordingProvider
	hook func()
}

func (p providerFunc) Create(name string, opts CreateOptions, meta *SandboxMeta) error {
	p.hook()
	return p.recordingProvider.Create(name, opts, meta)
}

func TestCreateRevokesTheAgentWhenProvisioningFails(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)
	stubEquip(t, errors.New("scp failed"))

	stderr := captureStderr(t)
	err := createExisting(t, "box")
	out := stderr()
	if err == nil {
		t.Fatal("create must fail when the equip step fails")
	}
	calls := host.calls()
	if len(calls) != 2 || calls[0] != "POST /api/agents/" || !strings.HasSuffix(calls[1], "/revoke/") {
		t.Fatalf("calls = %v, want a registration and then its revoke", calls)
	}
	got, _ := GetSandbox("box")
	if got == nil {
		t.Fatal("the machine must stay recorded, so provision or destroy can reach it")
	}
	if got.AgentID != "" {
		t.Errorf("a revoked agent is still recorded: %q", got.AgentID)
	}
	if !strings.Contains(out, "Revoked agent") {
		t.Errorf("create must say it revoked the agent:\n%s", out)
	}

	// The same name registers again: the new sandbox_id does not collide
	// with the one the revoked agent keeps.
	stubEquip(t, nil)
	stubStopAgent(t, nil)
	if err := Destroy(nil, "box", DestroyOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if err := createExisting(t, "box"); err != nil {
		t.Fatalf("create after a revoke: %v", err)
	}
}

func TestCreateRevokesTheAgentWhenTheProviderFails(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)

	captureStderr(t)
	p := &recordingProvider{ip: "9.9.9.9", dropletID: 7, createErr: errors.New("quota exceeded")}
	if err := Create(p, CreateOptions{Name: "box", Auth: AuthBedrock}); err == nil {
		t.Fatal("create must fail when the provider fails")
	}
	calls := host.calls()
	if len(calls) != 2 || calls[0] != "POST /api/agents/" || !strings.HasSuffix(calls[1], "/revoke/") {
		t.Fatalf("calls = %v, want a registration and then its revoke", calls)
	}
	if got, _ := GetSandbox("box"); got == nil || got.AgentID != "" {
		t.Errorf("record = %+v, want the machine recorded with no agent", got)
	}
}

func TestCreateKeepsTheAgentIDWhenTheRevokeFails(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)
	host.revokeStatus = 500
	stubEquip(t, errors.New("scp failed"))

	stderr := captureStderr(t)
	err := createExisting(t, "box")
	out := stderr()
	if err == nil {
		t.Fatal("create must fail when the equip step fails")
	}
	got, _ := GetSandbox("box")
	if got == nil || got.AgentID == "" || host.agent(got.AgentID) == nil {
		t.Fatalf("record = %+v, want the agent id kept, so destroy can revoke it", got)
	}
	if !strings.Contains(out, "could not revoke agent "+got.AgentID) {
		t.Errorf("create must warn with the agent id:\n%s", out)
	}
}

// Ctrl-C during a long create must not leave a live agent behind.
func TestCreateRevokesTheAgentOnInterrupt(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)
	codes := make(chan int, 1)
	origExit := exit
	exit = func(code int) { codes <- code }
	t.Cleanup(func() { exit = origExit })

	// The interrupt arrives while the machine is equipped. What the test
	// checks is the state at the moment the handler exits.
	var code int
	var callsAtExit []string
	var recordAtExit *SandboxMeta
	origEquip := equip
	equip = func(meta *SandboxMeta, name, agentToken string, mode AuthMode, artifacts *releaseArtifacts) *stageError {
		if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
			t.Fatal(err)
		}
		select {
		case code = <-codes:
		case <-time.After(5 * time.Second):
			t.Fatal("create did not exit after SIGINT")
		}
		callsAtExit = host.calls()
		recordAtExit, _ = GetSandbox(name)
		return &stageError{"deploying agent", errors.New("interrupted")}
	}
	t.Cleanup(func() { equip = origEquip })

	stderr := captureStderr(t)
	createExisting(t, "box")
	out := stderr()

	if code != 130 {
		t.Errorf("exit status = %d, want 130", code)
	}
	if len(callsAtExit) != 2 || !strings.HasSuffix(callsAtExit[1], "/revoke/") {
		t.Errorf("calls before exit = %v, want a registration and its revoke", callsAtExit)
	}
	if recordAtExit == nil || recordAtExit.AgentID != "" {
		t.Errorf("record at exit = %+v, want the machine with no agent", recordAtExit)
	}
	if !strings.Contains(out, "Interrupted") {
		t.Errorf("create must say it was interrupted:\n%s", out)
	}
}

func TestCreateWithoutAPITokenCallsNoControlHost(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)
	t.Setenv(apiTokenVar, "")
	tokens := stubEquip(t, nil)

	stderr := captureStderr(t)
	err := createExisting(t, "box")
	out := stderr()
	if err != nil {
		t.Fatal(err)
	}
	if calls := host.calls(); len(calls) != 0 {
		t.Errorf("create without an API token called the control host: %v", calls)
	}
	if len(*tokens) != 1 || !strings.Contains(out, (*tokens)[0]) {
		t.Errorf("the manual path must print the token it made:\n%s", out)
	}
	if !strings.Contains(out, "spekk sandbox register box") {
		t.Errorf("create must say how to register later:\n%s", out)
	}
}

// stubSSH records each remote command and answers with err.
func stubSSH(t *testing.T, err error) *[]string {
	t.Helper()
	var sent []string
	orig := sshExec
	sshExec = func(args []string) ([]byte, error) {
		sent = append(sent, args[len(args)-1])
		return nil, err
	}
	t.Cleanup(func() { sshExec = orig })
	return &sent
}

func TestRegister(t *testing.T) {
	t.Run("writes the token, keeps the record, and revokes the old agent", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		host := useControlHost(t)
		old := host.addAgent("box")
		if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4", Status: "active", AgentID: old}); err != nil {
			t.Fatal(err)
		}
		sent := stubSSH(t, nil)

		if err := Register("box", "org-1"); err != nil {
			t.Fatal(err)
		}
		got, _ := GetSandbox("box")
		agent := host.agent(got.AgentID)
		if got.AgentID == old || agent == nil {
			t.Fatalf("record = %+v, want the new agent", got)
		}
		if agent.Organization != "org-1" {
			t.Errorf("organization = %q, want org-1", agent.Organization)
		}
		if len(*sent) != 1 {
			t.Fatalf("sent %d SSH commands, want 1", len(*sent))
		}
		script := (*sent)[0]
		if strings.Contains(script, agent.Token) {
			t.Error("the token must travel base64-encoded, not as plain text")
		}
		// TestTokenScriptRestoresTheOldFile runs the script itself.
		if !strings.Contains(script, base64.StdEncoding.EncodeToString([]byte(agent.Token))) {
			t.Errorf("script lacks the encoded token:\n%s", script)
		}
		if !host.agent(old).Revoked {
			t.Error("the old agent stays live, but its token is no longer on the machine")
		}
	})

	t.Run("revokes the new agent when the machine cannot take the token", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		host := useControlHost(t)
		old := host.addAgent("box")
		if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4", Status: "active", AgentID: old}); err != nil {
			t.Fatal(err)
		}
		stubSSH(t, errors.New("connection refused"))

		if err := Register("box", ""); err == nil {
			t.Fatal("register must fail when SSH fails")
		}
		calls := host.calls()
		if len(calls) != 2 || !strings.HasSuffix(calls[1], "/revoke/") {
			t.Errorf("calls = %v, want a registration and its revoke", calls)
		}
		if got, _ := GetSandbox("box"); got.AgentID != old || host.agent(old).Revoked {
			t.Errorf("record names %q, want the old agent %q, still live", got.AgentID, old)
		}
	})
}

// The token script runs here in a local shell, against a temporary env
// file, with stand-ins for chown and systemctl. A failed restart must put
// the old file back, so the machine keeps the agent the record names.
func TestTokenScriptRestoresTheOldFile(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	const old = "CLAUDE_CODE_OAUTH_TOKEN=keep-me\nSPEKK_AGENT_TOKEN=old-token\n"
	run := func(t *testing.T, restart int) (dir string, log string, err error) {
		dir = t.TempDir()
		bin := filepath.Join(dir, "bin")
		if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "etc", "agent.env"), []byte(old), 0o600); err != nil {
			t.Fatal(err)
		}
		os.MkdirAll(bin, 0o755)
		os.WriteFile(filepath.Join(bin, "chown"), []byte("#!/bin/sh\nexit 0\n"), 0o755)
		// systemctl logs the token it would start with, then answers restart.
		systemctl := fmt.Sprintf("#!/bin/sh\ngrep SPEKK_AGENT_TOKEN %s/etc/agent.env >> %s/log\nexit %d\n", dir, dir, restart)
		os.WriteFile(filepath.Join(bin, "systemctl"), []byte(systemctl), 0o755)

		script := strings.ReplaceAll(buildTokenScript("new-token"), "/etc/spekk", filepath.Join(dir, "etc"))
		cmd := exec.Command("sh", "-c", script)
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
		_, err = cmd.CombinedOutput()
		data, _ := os.ReadFile(filepath.Join(dir, "log"))
		return dir, string(data), err
	}
	read := func(t *testing.T, path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	t.Run("a failed restart puts the old file back", func(t *testing.T) {
		dir, log, err := run(t, 1)
		if err == nil {
			t.Fatal("the script must fail when the restart fails")
		}
		if got := read(t, filepath.Join(dir, "etc", "agent.env")); got != old {
			t.Errorf("agent.env = %q, want the old file %q", got, old)
		}
		if !strings.HasSuffix(log, "SPEKK_AGENT_TOKEN=old-token\n") {
			t.Errorf("the last restart must use the old token, log:\n%s", log)
		}
		for _, f := range []string{"agent.env.bak", "agent.env.new"} {
			if _, err := os.Stat(filepath.Join(dir, "etc", f)); err == nil {
				t.Errorf("%s is left behind", f)
			}
		}
	})

	t.Run("success writes the new token and removes the copy", func(t *testing.T) {
		dir, _, err := run(t, 0)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "etc", "agent.env")
		if got := read(t, path); got != "CLAUDE_CODE_OAUTH_TOKEN=keep-me\nSPEKK_AGENT_TOKEN=new-token\n" {
			t.Errorf("agent.env = %q", got)
		}
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v, want 600", info.Mode().Perm())
		}
		if _, err := os.Stat(path + ".bak"); err == nil {
			t.Error("agent.env.bak is left behind")
		}
	})
}

func TestProvisionAgentToken(t *testing.T) {
	setup := func(t *testing.T, agentID string) (*fakeControlHost, *[]string) {
		isolateConfig(t)
		useTempStore(t)
		stubCreateEnv(t)
		host := useControlHost(t)
		if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4", Status: "active", AgentID: agentID}); err != nil {
			t.Fatal(err)
		}
		orig := checkReady
		checkReady = func(meta *SandboxMeta, name string) error { return nil }
		t.Cleanup(func() { checkReady = orig })
		return host, stubEquip(t, nil)
	}
	stubRead := func(t *testing.T, token string, err error) {
		orig := readAgentToken
		readAgentToken = func(meta *SandboxMeta, name string) (string, error) { return token, err }
		t.Cleanup(func() { readAgentToken = orig })
	}

	t.Run("a registered sandbox keeps its token and its agent", func(t *testing.T) {
		host, tokens := setup(t, "agent-1")
		stubRead(t, "token-on-the-machine", nil)
		if err := Provision("box", ProvisionOptions{Force: true}); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(*tokens, []string{"token-on-the-machine"}) {
			t.Errorf("equip got %v, want the token already on the machine", *tokens)
		}
		if calls := host.calls(); len(calls) != 0 {
			t.Errorf("provision of a registered sandbox called the control host: %v", calls)
		}
		if got, _ := GetSandbox("box"); got.AgentID != "agent-1" {
			t.Errorf("AgentID = %q, want agent-1", got.AgentID)
		}
	})

	t.Run("an unreadable token stops before the machine changes", func(t *testing.T) {
		_, tokens := setup(t, "agent-1")
		stubRead(t, "", errors.New("no SPEKK_AGENT_TOKEN"))
		err := Provision("box", ProvisionOptions{Force: true})
		if err == nil || !strings.Contains(err.Error(), "spekk sandbox register box") {
			t.Fatalf("want an error that names register, got %v", err)
		}
		if len(*tokens) != 0 {
			t.Error("the machine was equipped with no token to give it")
		}
	})

	t.Run("a failed SSH read is not reported as a missing token", func(t *testing.T) {
		setup(t, "agent-1")
		// A stand-in ssh: the first answer is a refused connection, the
		// second an env file with no token.
		bin := t.TempDir()
		script := "#!/bin/sh\nif [ -n \"$SSH_REFUSE\" ]; then echo 'ssh: connect to host 1.2.3.4 port 22: Connection refused' >&2; exit 255; fi\nexit 0\n"
		if err := os.WriteFile(filepath.Join(bin, "ssh"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		meta := &SandboxMeta{IP: "1.2.3.4"}

		t.Setenv("SSH_REFUSE", "1")
		_, sshErr := readAgentToken(meta, "box")
		t.Setenv("SSH_REFUSE", "")
		_, emptyErr := readAgentToken(meta, "box")
		if sshErr == nil || !strings.Contains(sshErr.Error(), "Connection refused") {
			t.Errorf("SSH failure error = %v, want the ssh message", sshErr)
		}
		if emptyErr == nil || !strings.Contains(emptyErr.Error(), "no SPEKK_AGENT_TOKEN") || strings.Contains(emptyErr.Error(), "SSH") {
			t.Errorf("missing token error = %v, want only the missing token", emptyErr)
		}
	})

	t.Run("an unregistered sandbox registers when it can", func(t *testing.T) {
		host, tokens := setup(t, "")
		if err := Provision("box", ProvisionOptions{Force: true}); err != nil {
			t.Fatal(err)
		}
		got, _ := GetSandbox("box")
		agent := host.agent(got.AgentID)
		if agent == nil || !slices.Equal(*tokens, []string{agent.Token}) {
			t.Errorf("record %+v and token %v do not match a registered agent", got, *tokens)
		}
	})
}

// A provision that registered an agent and then cannot save the record
// must name the agent, because nothing else does.
func TestProvisionNamesTheAgentWhenTheSaveFails(t *testing.T) {
	isolateConfig(t)
	store := useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)
	if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	origReady := checkReady
	checkReady = func(meta *SandboxMeta, name string) error { return nil }
	t.Cleanup(func() { checkReady = origReady })
	orig := equip
	equip = func(meta *SandboxMeta, name, agentToken string, mode AuthMode, artifacts *releaseArtifacts) *stageError {
		// The machine is equipped, then the record becomes unreadable.
		if err := os.WriteFile(store, []byte("not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		return nil
	}
	t.Cleanup(func() { equip = orig })

	err := Provision("box", ProvisionOptions{Force: true})
	if err == nil {
		t.Fatal("provision must fail when the record cannot be saved")
	}
	calls := host.calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want the registration only (error: %v)", calls, err)
	}
	var id string
	for k := range host.agents {
		id = k
	}
	if !strings.Contains(err.Error(), id) {
		t.Errorf("error %q does not name agent %s", err, id)
	}
}

func TestDestroyRevokesFirst(t *testing.T) {
	setup := func(t *testing.T) (*fakeControlHost, string) {
		isolateConfig(t)
		useTempStore(t)
		host := useControlHost(t)
		id := host.addAgent("box")
		if err := SaveSandbox("box", &SandboxMeta{Provider: "digitalocean", DropletID: 7, IP: "1.2.3.4", AgentID: id}); err != nil {
			t.Fatal(err)
		}
		return host, id
	}

	t.Run("revokes, then destroys", func(t *testing.T) {
		host, id := setup(t)
		if err := Destroy(&recordingProvider{}, "box", DestroyOptions{Force: true}); err != nil {
			t.Fatal(err)
		}
		if !host.agent(id).Revoked {
			t.Error("the agent is still live on the control host")
		}
		if got, _ := GetSandbox("box"); got != nil {
			t.Error("the record should be gone")
		}
	})

	t.Run("a failed revoke stops the destroy, even with --force", func(t *testing.T) {
		host, id := setup(t)
		host.revokeStatus = 403
		p := &destroyCounter{}
		err := Destroy(p, "box", DestroyOptions{Force: true})
		if err == nil || !strings.Contains(err.Error(), id) || !strings.Contains(err.Error(), "--ignore-revoke-failure") {
			t.Fatalf("want an error naming the agent and --ignore-revoke-failure, got %v", err)
		}
		if p.calls != 0 {
			t.Error("the machine was destroyed while its agent is still live")
		}
		if got, _ := GetSandbox("box"); got == nil {
			t.Error("the record must stay")
		}
	})

	t.Run("--ignore-revoke-failure destroys after a failed revoke", func(t *testing.T) {
		host, id := setup(t)
		host.revokeStatus = 403
		p := &destroyCounter{}
		stderr := captureStderr(t)
		err := Destroy(p, "box", DestroyOptions{Force: true, IgnoreRevokeFailure: true})
		out := stderr()
		if err != nil {
			t.Fatal(err)
		}
		if p.calls != 1 {
			t.Error("--ignore-revoke-failure should destroy the machine")
		}
		if !strings.Contains(out, id) {
			t.Errorf("the warning must name the agent:\n%s", out)
		}
	})

	t.Run("no API token stops before the prompt", func(t *testing.T) {
		host, id := setup(t)
		t.Setenv(apiTokenVar, "")
		p := &destroyCounter{}
		err := Destroy(p, "box", DestroyOptions{})
		if err == nil || !strings.Contains(err.Error(), id) || !strings.Contains(err.Error(), apiTokenVar) {
			t.Fatalf("want an error naming the agent and %s, got %v", apiTokenVar, err)
		}
		if p.calls != 0 || len(host.calls()) != 0 {
			t.Error("destroy went on without the API token")
		}
		if got, _ := GetSandbox("box"); got == nil {
			t.Error("the record must stay")
		}

		if err := Destroy(p, "box", DestroyOptions{Force: true, IgnoreRevokeFailure: true}); err != nil {
			t.Fatal(err)
		}
		if p.calls != 1 {
			t.Error("--ignore-revoke-failure should destroy the machine")
		}
	})
}

// destroyCounter counts the destroys it is asked for.
type destroyCounter struct {
	recordingProvider
	calls int
}

func (d *destroyCounter) Destroy(meta *SandboxMeta) error {
	d.calls++
	return nil
}
