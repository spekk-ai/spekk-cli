package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// validMachineInput is a create stdin with model credential mc, for a
// control host at host.
func validMachineInput(host, mc string) string {
	return `{"digitalocean_token": "do-secret", "registration_token": "reg-secret", "spekk_host": "` + host +
		`", "github_token": "ghs-secret", "github_repository": "acme/widgets", "harness": "claude-code", "model_credential": ` + mc + `}`
}

func TestReadMachineInput(t *testing.T) {
	const host = "spekk.example.com"
	cases := []struct {
		name, stdin string
		auth        AuthMode // empty: the input is refused
	}{
		{"none", validMachineInput(host, `{"kind": "none"}`), AuthNone},
		{"api key", validMachineInput(host, `{"kind": "anthropic_api_key", "api_key": "sk-secret"}`), AuthAPIKey},
		{"bedrock", validMachineInput(host, `{"kind": "bedrock", "aws_access_key_id": "a", "aws_secret_access_key": "b", "aws_region": "us-east-1"}`), AuthBedrock},
		{"missing field", `{"digitalocean_token": "x", "registration_token": "x", "spekk_host": "h", "github_token": "x", "model_credential": {"kind": "none"}}`, ""},
		{"empty field", strings.Replace(validMachineInput(host, `{"kind": "none"}`), `"reg-secret"`, `""`, 1), ""},
		{"unknown field", strings.Replace(validMachineInput(host, `{"kind": "none"}`), `{`, `{"aws_access_key_id": "x", `, 1), ""},
		{"unknown credential field", validMachineInput(host, `{"kind": "none", "extra": "x"}`), ""},
		{"field of another type", validMachineInput(host, `{"kind": "bedrock", "api_key": "x", "aws_access_key_id": "a", "aws_secret_access_key": "b", "aws_region": "r"}`), ""},
		{"missing credential field", validMachineInput(host, `{"kind": "anthropic_api_key"}`), ""},
		{"subscription token", validMachineInput(host, `{"kind": "subscription", "oauth_token": "x"}`), ""},
		{"no model credential", strings.Replace(validMachineInput(host, `{"kind": "none"}`), `, "model_credential": {"kind": "none"}`, ``, 1), ""},
		{"unknown harness", strings.Replace(validMachineInput(host, `{"kind": "none"}`), `"claude-code"`, `"other-harness"`, 1), ""},
		{"trailing data", validMachineInput(host, `{"kind": "none"}`) + `{}`, ""},
		{"bad repository", strings.Replace(validMachineInput(host, `{"kind": "none"}`), "acme/widgets", "acme/../x", 1), ""},
		{"dot repository", strings.Replace(validMachineInput(host, `{"kind": "none"}`), "acme/widgets", "acme/..", 1), ""},
		{"http host", validMachineInput("http://spekk.example.com", `{"kind": "none"}`), ""},
		{"too large", validMachineInput(host, `{"kind": "none"}`) + strings.Repeat(" ", maxMachineInput), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in, err := readMachineInput(strings.NewReader(tc.stdin))
			if tc.auth == "" {
				if err == nil {
					t.Fatalf("accepted %s", tc.stdin)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if in.auth != tc.auth || in.controlHost != "https://spekk.example.com" || in.repository != "acme/widgets" {
				t.Fatalf("read %+v", in)
			}
		})
	}
}

// fakeDigitalOcean answers the calls of a droplet create and destroy, and
// records each request as "METHOD path" with its Authorization header.
type fakeDigitalOcean struct {
	mu       sync.Mutex
	requests []string
	tags     []string
	auth     []string
}

func useFakeDigitalOcean(t *testing.T) *fakeDigitalOcean {
	t.Helper()
	f := &fakeDigitalOcean{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/v2/account/keys":
			fmt.Fprint(w, `{"ssh_key": {"id": 11}}`)
		case r.Method == "POST" && r.URL.Path == "/v2/droplets":
			fmt.Fprint(w, `{"droplet": {"id": 4242}}`)
		case r.Method == "GET" && r.URL.Path == "/v2/droplets/4242":
			fmt.Fprint(w, `{"droplet": {"id": 4242, "status": "active", "networks": {"v4": [{"ip_address": "203.0.113.10", "type": "public"}]}}}`)
		case r.Method == "POST" && r.URL.Path == "/v2/tags":
			var tag struct{ Name string }
			json.Unmarshal(body, &tag)
			f.tags = append(f.tags, tag.Name)
			w.WriteHeader(201)
			fmt.Fprint(w, `{}`)
		default:
			w.WriteHeader(204)
		}
	}))
	t.Cleanup(srv.Close)
	orig := machineDOClient
	machineDOClient = func(token string) *Client { return NewClientWithToken(token, srv.URL) }
	t.Cleanup(func() { machineDOClient = orig })
	return f
}

func (f *fakeDigitalOcean) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

// registration is one request to the fake register endpoint.
type registration struct {
	auth string
	body map[string]string
}

// useRegisterHost starts a control host that answers POST
// /api/agents/register/ with status, and records each request.
func useRegisterHost(t *testing.T, status int) (string, *[]registration) {
	t.Helper()
	var got []registration
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		got = append(got, registration{r.Header.Get("Authorization"), body})
		if r.Method != "POST" || r.URL.Path != "/api/agents/register/" {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(status)
		if status == 201 {
			fmt.Fprint(w, `{"id": "agent-77"}`)
		} else {
			fmt.Fprint(w, `{"detail": "Invalid or expired registration token."}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &got
}

// stageLines decodes machine-mode stdout, and fails on a line that is not
// a stage line.
func stageLines(t *testing.T, stdout string) []stageLine {
	t.Helper()
	var lines []stageLine
	for _, raw := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		var l stageLine
		if err := json.Unmarshal([]byte(raw), &l); err != nil || l.Stage == "" {
			t.Fatalf("stdout has a line that is not a stage line: %q", raw)
		}
		lines = append(lines, l)
	}
	return lines
}

// machineSSH stubs the steps of a machine-mode create that reach the
// machine, and records what each ssh call sent. failOn makes the call whose
// script holds it fail.
func machineSSH(t *testing.T, failOn string) (argv, stdin *[]string) {
	t.Helper()
	argv, stdin = &[]string{}, &[]string{}
	origSSH, origSCP, origFetch, origWait := sshExec, scpExec, fetchAgent, waitReady
	sshExec = func(args []string, in string) ([]byte, error) {
		*argv = append(*argv, strings.Join(args, " "))
		*stdin = append(*stdin, in)
		if failOn != "" && strings.Contains(sentScript(args, in), failOn) {
			return []byte("Job for spekk-agent.service failed"), fmt.Errorf("exit status 1")
		}
		if strings.Contains(args[len(args)-1], claudeVersionLine) {
			return []byte("2.1.301 (Claude Code)\n"), nil
		}
		return nil, nil
	}
	scpExec = func([]string) ([]byte, error) { return nil, nil }
	fetchAgent = func(*SandboxMeta, string, *releaseArtifacts) error { return nil }
	waitReady = func(ip, keyPath, name, user string, timeout time.Duration) error { return nil }
	t.Cleanup(func() { sshExec, scpExec, fetchAgent, waitReady = origSSH, origSCP, origFetch, origWait })
	return argv, stdin
}

// machineOptions is a machine-mode create as the server runs it.
func machineOptions() CreateOptions {
	return CreateOptions{Name: "launch-1", Size: "s-2vcpu-4gb", Region: "nyc3", Release: "v1.99.0",
		Kind: KindOneOff, ExpiresAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
		Labels: map[string]string{"spekk-launch": "abc123"}}
}

func TestCreateMachine(t *testing.T) {
	setUp := func(t *testing.T, registerStatus int, failOn string) (do *fakeDigitalOcean, regs *[]registration, argv, stdin *[]string, input string) {
		isolateConfig(t)
		useTempStore(t)
		stubCreateEnv(t)
		stubHostname(t)
		// Machine mode reads no secret from its environment.
		t.Setenv("DO_API_TOKEN", "env-do-token")
		t.Setenv("SPEKK_HOST", "env-host.invalid")
		do = useFakeDigitalOcean(t)
		host, regs := useRegisterHost(t, registerStatus)
		argv, stdin = machineSSH(t, failOn)
		input = validMachineInput(host, `{"kind": "anthropic_api_key", "api_key": "sk-ant-secret"}`)
		return do, regs, argv, stdin, input
	}
	secrets := []string{"do-secret", "reg-secret", "ghs-secret", "sk-ant-secret"}

	t.Run("online", func(t *testing.T) {
		do, regs, argv, stdin, input := setUp(t, 201, "")
		var stdout strings.Builder
		stderr := captureStderr(t)
		if err := CreateMachine(strings.NewReader(input), &stdout, machineOptions()); err != nil {
			t.Fatal(err)
		}
		errOut := stderr()

		lines := stageLines(t, stdout.String())
		want := []stageLine{{Stage: "creating"}, {Stage: "provisioning", DropletID: 4242, IP: "203.0.113.10"}, {Stage: "registering"}, {Stage: "online", AgentID: "agent-77"}}
		if fmt.Sprint(lines) != fmt.Sprint(want) {
			t.Fatalf("stage lines = %+v, want %+v", lines, want)
		}

		for _, a := range do.auth {
			if a != "Bearer do-secret" {
				t.Errorf("a DigitalOcean request has Authorization %q, want the token from stdin", a)
			}
		}
		if got := strings.Join(do.tags, " "); got != "spekk-expires:2026-10-20 spekk-host:ops-laptop spekk-kind:one-off spekk-launch:abc123" {
			t.Errorf("droplet tags = %s", got)
		}

		if len(*regs) != 1 {
			t.Fatalf("got %d registrations, want 1", len(*regs))
		}
		reg := (*regs)[0]
		agentToken := reg.body["token"]
		if reg.auth != "Bearer reg-secret" || reg.body["name"] != "launch-1" || !strings.HasPrefix(reg.body["sandbox_id"], "spekk-launch-1-") || len(agentToken) < 32 || len(reg.body) != 3 {
			t.Errorf("registration = %+v", reg)
		}

		sent := strings.Join(*stdin, "\n")
		if !holdsSecret(sent, "SPEKK_AGENT_TOKEN="+agentToken+"\n") || !holdsSecret(sent, "SPEKK_AGENT_ID=agent-77\n") || !holdsSecret(sent, "ANTHROPIC_API_KEY=sk-ant-secret\n") {
			t.Error("the env file on the machine does not have the agent token, the agent id, and the API key")
		}
		if !holdsSecret(sent, "ghs-secret") || !strings.Contains(sent, "'https://github.com/acme/widgets.git'") {
			t.Error("the clone script did not reach the machine on stdin")
		}
		for _, secret := range append(secrets, agentToken) {
			for _, a := range *argv {
				if holdsSecret(a, secret) {
					t.Errorf("an ssh argument list holds %s: %s", secret, a)
				}
			}
			if strings.Contains(stdout.String(), secret) || strings.Contains(errOut, secret) {
				t.Errorf("stdout or stderr holds %s", secret)
			}
		}

		got, _ := GetSandbox("launch-1")
		if got == nil || got.Status != "active" || got.DropletID != 4242 || got.AgentID != "agent-77" || got.Auth != "api-key" || got.SSHKeyPath == "" {
			t.Fatalf("record = %+v", got)
		}
	})

	t.Run("a failed deploy keeps the droplet on record", func(t *testing.T) {
		_, _, _, _, input := setUp(t, 201, "systemctl restart spekk-agent")
		var stdout strings.Builder
		stderr := captureStderr(t)
		err := CreateMachine(strings.NewReader(input), &stdout, machineOptions())
		errOut := stderr()
		if err == nil {
			t.Fatal("create succeeded with a failed deploy")
		}
		lines := stageLines(t, stdout.String())
		last := lines[len(lines)-1]
		if last.Stage != "failed" || last.At != "registering" || !strings.Contains(last.Message, "deploying agent") {
			t.Fatalf("last line = %+v", last)
		}
		for _, secret := range secrets {
			if strings.Contains(stdout.String(), secret) || strings.Contains(errOut, secret) || strings.Contains(err.Error(), secret) {
				t.Errorf("the output holds %s", secret)
			}
		}
		if got, _ := GetSandbox("launch-1"); got == nil || got.DropletID != 4242 || got.AgentID != "agent-77" {
			t.Fatalf("record = %+v, want the droplet and the agent", got)
		}
	})

	t.Run("a refused registration stops before the machine is equipped", func(t *testing.T) {
		_, _, _, stdin, input := setUp(t, 401, "")
		var stdout strings.Builder
		if err := CreateMachine(strings.NewReader(input), &stdout, machineOptions()); err == nil {
			t.Fatal("create succeeded with a refused registration")
		}
		lines := stageLines(t, stdout.String())
		last := lines[len(lines)-1]
		if last.Stage != "failed" || last.At != "registering" || !strings.Contains(last.Message, "401") || !strings.Contains(last.Message, "Invalid or expired registration token.") {
			t.Fatalf("last line = %+v", last)
		}
		if len(*stdin) != 0 {
			t.Errorf("%d scripts went to the machine after a refused registration", len(*stdin))
		}
	})

	t.Run("bad input stops before any call", func(t *testing.T) {
		do, regs, _, _, _ := setUp(t, 201, "")
		var stdout strings.Builder
		err := CreateMachine(strings.NewReader(`{"digitalocean_token": "do-secret"}`), &stdout, machineOptions())
		if err == nil {
			t.Fatal("create accepted a bad input")
		}
		if lines := stageLines(t, stdout.String()); len(lines) != 1 || lines[0].Stage != "failed" || lines[0].At != "creating" {
			t.Fatalf("stage lines = %+v", lines)
		}
		if len(do.calls()) != 0 || len(*regs) != 0 {
			t.Errorf("a bad input reached DigitalOcean %v or the control host %v", do.calls(), *regs)
		}
		if got, _ := GetSandbox("launch-1"); got != nil {
			t.Errorf("a bad input left a record: %+v", got)
		}
	})
}

func TestDestroyMachine(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	controlHost := useControlHost(t)
	do := useFakeDigitalOcean(t)
	key := generatedKeysDir() + "/launch-1"
	if err := os.MkdirAll(generatedKeysDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{key, key + ".pub"} {
		if err := os.WriteFile(f, []byte("key"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := SaveSandbox("launch-1", &SandboxMeta{Provider: "digitalocean", DropletID: 4242, SSHKeyID: 11, SSHKeyPath: key, IP: "203.0.113.10", AgentID: "agent-77"}); err != nil {
		t.Fatal(err)
	}

	if err := DestroyMachine(strings.NewReader(`{"digitalocean_token": "do-secret"}`), "launch-1"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(do.calls(), ", "); got != "DELETE /v2/droplets/4242, DELETE /v2/account/keys/11" {
		t.Errorf("DigitalOcean calls = %s", got)
	}
	for _, a := range do.auth {
		if a != "Bearer do-secret" {
			t.Errorf("a DigitalOcean request has Authorization %q, want the token from stdin", a)
		}
	}
	if calls := controlHost.calls(); len(calls) != 0 {
		t.Errorf("destroy --machine called the control host: %v", calls)
	}
	if got, _ := GetSandbox("launch-1"); got != nil {
		t.Errorf("the record is still there: %+v", got)
	}
	if _, err := os.Stat(key); !os.IsNotExist(err) {
		t.Errorf("the key pair is still there: %v", err)
	}
}
