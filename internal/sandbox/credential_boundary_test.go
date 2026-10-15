package sandbox

import (
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
