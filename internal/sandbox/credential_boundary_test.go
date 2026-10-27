package sandbox

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The control host never receives a cloud or model credential of the user.
// Every command that calls it runs here, with each such credential set to a
// value that is easy to find, and no request may carry one. See
// specs/sandbox-self-register/assertions/no-cloud-credential-to-control-host.md.
func TestNoCloudCredentialReachesTheControlHost(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	host := useControlHost(t)

	secrets := map[string]string{}
	for _, v := range []string{"DO_API_TOKEN", "DIGITALOCEAN_TOKEN", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "GOOGLE_APPLICATION_CREDENTIALS", oauthTokenVar, "ANTHROPIC_API_KEY", "GITHUB_TOKEN"} {
		secrets[v] = "secret-of-" + strings.ToLower(v)
		t.Setenv(v, secrets[v])
	}
	stubEquip(t, nil)
	stubSSH(t, nil)
	stubStopAgent(t, nil)
	orig := readAgentToken
	readAgentToken = func(meta *SandboxMeta, name string) (string, error) { return "token-on-the-machine", nil }
	t.Cleanup(func() { readAgentToken = orig })

	steps := []struct {
		name string
		run  func() error
	}{
		{"create", func() error { return createExisting(t, "box") }},
		{"register", func() error { return Register("box", "") }},
		{"provision", func() error { return Provision("box", ProvisionOptions{Force: true}) }},
		{"schedule create", func() error {
			_, err := runSchedule(t, "create", "box", "--slug", "job", "--cron_expression", "0 9 * * *", "--task_text", "x")
			return err
		}},
		{"schedule list", func() error { _, err := runSchedule(t, "list", "box"); return err }},
		{"schedule update", func() error { _, err := runSchedule(t, "update", "box", "job", "--role", "coach"); return err }},
		{"schedule run", func() error { _, err := runSchedule(t, "run", "box", "job"); return err }},
		{"schedule delete", func() error { _, err := runSchedule(t, "delete", "box", "job"); return err }},
		{"destroy", func() error { return Destroy(nil, "box", DestroyOptions{Force: true}) }},
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
	}
	if len(host.requests) < len(steps) {
		t.Fatalf("only %d requests reached the control host; a step did not run", len(host.requests))
	}

	for _, r := range host.requests {
		where := r.Method + " " + r.Path
		var headers strings.Builder
		for k, vs := range r.Header {
			headers.WriteString(k + ": " + strings.Join(vs, ",") + "\n")
		}
		for v, secret := range secrets {
			for part, text := range map[string]string{"URL": r.Path + "?" + r.RawQuery, "headers": headers.String(), "body": r.Body} {
				if strings.Contains(text, secret) {
					t.Errorf("%s carries %s in its %s", where, v, part)
				}
			}
		}
		if got := r.Header.Values("Authorization"); len(got) != 1 || got[0] != "Token "+testAPIToken {
			t.Errorf("%s has Authorization %q, want only the API token", where, got)
		}
		for k := range r.Header {
			lower := strings.ToLower(k)
			if k != "Authorization" && (strings.Contains(lower, "auth") || strings.Contains(lower, "token")) {
				t.Errorf("%s has an unexpected credential header %s", where, k)
			}
		}
	}
}

// base64Run finds the runs of text that may be base64.
var base64Run = regexp.MustCompile(`[A-Za-z0-9+/]{8,}={0,2}`)

// holdsSecret reports whether text holds secret in clear text, or in a
// base64 run that decodes to text that holds it. sudoWrap encodes a script
// that already holds base64, so the search goes down each level.
func holdsSecret(text, secret string) bool {
	if strings.Contains(text, secret) {
		return true
	}
	for _, run := range base64Run.FindAllString(text, -1) {
		if decoded, err := base64.StdEncoding.DecodeString(run); err == nil && holdsSecret(string(decoded), secret) {
			return true
		}
	}
	return false
}

// Each process on the host can read the argument list of ssh in
// /proc/<pid>/cmdline. Each step that sends a secret to the machine runs
// here, for root and for a sudo login, and no ssh argument may hold one,
// in clear text or in base64. The secrets must reach the machine on stdin,
// so the test also fails when a step stops sending them. See
// specs/sandbox-machine-mode/assertions/secrets-on-ssh-stdin.md.
func TestNoSecretInTheSSHArgumentList(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	host := useControlHost(t)

	secrets := map[string]string{"agent token": "agent-token-for-the-machine"}
	for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", oauthTokenVar, "GITHUB_TOKEN"} {
		secrets[v] = "secret-of-" + strings.ToLower(v)
		t.Setenv(v, secrets[v])
	}
	codexPath := filepath.Join(t.TempDir(), "auth.json")
	codexSecret := `{"auth_mode":"chatgpt","tokens":{"refresh_token":"secret-of-codex-refresh"}}`
	if err := os.WriteFile(codexPath, []byte(codexSecret), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(codexAuthFileVar, codexPath)
	secrets["codex refresh"] = "secret-of-codex-refresh"
	var argv []string
	var stdin strings.Builder
	orig := sshExec
	sshExec = func(args []string, in string) ([]byte, error) {
		argv = append(argv, strings.Join(args, " "))
		stdin.WriteString(in)
		return nil, nil
	}
	t.Cleanup(func() { sshExec = orig })

	for _, user := range []string{"root", "ubuntu"} {
		for _, mode := range []AuthMode{AuthBedrock, AuthSubscription} {
			if err := injectCredentials("9.9.9.9", "", "box", user, secrets["agent token"], "agent-1", mode, nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := configureGitCredentials("9.9.9.9", "", "box", user); err != nil {
			t.Fatal(err)
		}
		if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "9.9.9.9", SSHUser: user}); err != nil {
			t.Fatal(err)
		}
		if err := Register("box", ""); err != nil {
			t.Fatal(err)
		}
		got, _ := GetSandbox("box")
		secrets["registered agent token as "+user] = host.agent(got.AgentID).Token
	}

	for name, secret := range secrets {
		for _, args := range argv {
			if holdsSecret(args, secret) {
				t.Errorf("an ssh argument list holds the %s: %s", name, args)
			}
		}
		if !holdsSecret(stdin.String(), secret) {
			t.Errorf("the %s did not reach the machine on stdin", name)
		}
	}
}
