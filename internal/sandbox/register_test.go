package sandbox

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	if err := Destroy(nil, "box", true); err != nil {
		t.Fatal(err)
	}
	if err := createExisting(t, "box"); err != nil {
		t.Fatalf("create after a revoke: %v", err)
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
		for _, want := range []string{base64.StdEncoding.EncodeToString([]byte(agent.Token)), "grep -v '^SPEKK_AGENT_TOKEN='", "chmod 600", "systemctl restart spekk-agent"} {
			if !strings.Contains(script, want) {
				t.Errorf("script lacks %q:\n%s", want, script)
			}
		}
		if !host.agent(old).Revoked {
			t.Error("the old agent stays live, but its token is no longer on the machine")
		}
	})

	t.Run("revokes the new agent when the machine cannot take the token", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		host := useControlHost(t)
		if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4", Status: "active"}); err != nil {
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
		if got, _ := GetSandbox("box"); got.AgentID != "" {
			t.Errorf("a revoked agent was recorded: %q", got.AgentID)
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
		if err := Destroy(&recordingProvider{}, "box", true); err != nil {
			t.Fatal(err)
		}
		if !host.agent(id).Revoked {
			t.Error("the agent is still live on the control host")
		}
		if got, _ := GetSandbox("box"); got != nil {
			t.Error("the record should be gone")
		}
	})

	t.Run("a failed revoke stops the destroy", func(t *testing.T) {
		host, id := setup(t)
		host.revokeStatus = 403
		answerPrompt(t, "y\n")
		p := &destroyCounter{}
		err := Destroy(p, "box", false)
		if err == nil || !strings.Contains(err.Error(), id) || !strings.Contains(err.Error(), "--force") {
			t.Fatalf("want an error naming the agent and --force, got %v", err)
		}
		if p.calls != 0 {
			t.Error("the machine was destroyed while its agent is still live")
		}
		if got, _ := GetSandbox("box"); got == nil {
			t.Error("the record must stay")
		}
	})

	t.Run("--force destroys after a failed revoke", func(t *testing.T) {
		host, _ := setup(t)
		host.revokeStatus = 403
		p := &destroyCounter{}
		if err := Destroy(p, "box", true); err != nil {
			t.Fatal(err)
		}
		if p.calls != 1 {
			t.Error("--force should destroy the machine")
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
