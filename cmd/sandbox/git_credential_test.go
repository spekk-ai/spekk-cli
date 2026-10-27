package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useAgentEnvFile writes an env file for the agent of host, as machine mode
// writes it.
func useAgentEnvFile(t *testing.T, host string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.env")
	content := "SPEKK_HOST=" + strings.TrimPrefix(host, "http://") + "\nSPEKK_AGENT_TOKEN=agent-secret\nSPEKK_AGENT_ID=agent-77\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := agentEnvFile
	agentEnvFile = path
	t.Cleanup(func() { agentEnvFile = orig })
}

func TestGitCredentialHelper(t *testing.T) {
	var requests []string
	status, answer := 200, `{"token": "ghs-new", "expires_at": "2026-10-17T13:00:00Z"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
		w.WriteHeader(status)
		w.Write([]byte(answer))
	}))
	defer srv.Close()
	useAgentEnvFile(t, srv.URL)

	run := func(action, request string) (code int, stdout, stderr string) {
		var out, errOut strings.Builder
		code = runGitCredential([]string{action}, strings.NewReader(request), &out, &errOut)
		return code, out.String(), errOut.String()
	}
	github := "protocol=https\nhost=github.com\npath=acme/widgets.git\n\n"

	code, out, _ := run("get", github)
	if code != 0 || out != "username=x-access-token\npassword=ghs-new\n" {
		t.Errorf("get = %d %q", code, out)
	}
	if len(requests) != 1 || requests[0] != "POST /api/agents/agent-77/git-token/ Bearer agent-secret" {
		t.Errorf("requests = %q", requests)
	}

	// git shows the stderr of a helper in its error, so the cause reaches
	// the agent.
	status, answer = 409, `{"detail": "The GitHub App no longer has access to acme/widgets"}`
	code, out, errOut := run("get", github)
	if code != 1 || out != "" || !strings.Contains(errOut, "409: The GitHub App no longer has access to acme/widgets") || strings.Contains(errOut, "agent-secret") {
		t.Errorf("refused get = %d %q %q", code, out, errOut)
	}

	requests = nil
	for _, c := range []struct{ action, request string }{
		{"get", "protocol=https\nhost=gitlab.com\n\n"},
		{"get", "protocol=http\nhost=github.com\n\n"},
		{"store", github + "username=x\npassword=y\n"},
		{"erase", github},
	} {
		if code, out, _ := run(c.action, c.request); code != 0 || out != "" {
			t.Errorf("%s %q = %d %q, want nothing", c.action, c.request, code, out)
		}
	}
	if len(requests) != 0 {
		t.Errorf("a request that is not a GitHub get reached the control host: %q", requests)
	}
}

// The service and the helper run as one user, so a file that the service
// cannot read means that each repository chat fails. The start warning
// tells the operator before a turn does.
func TestAgentEnvWarning(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a file of any mode")
	}
	path := filepath.Join(t.TempDir(), "agent.env")
	if err := os.WriteFile(path, []byte("SPEKK_HOST=h\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := agentEnvWarning(path); got != "" {
		t.Errorf("a readable file gave a warning: %s", got)
	}

	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	got := agentEnvWarning(path)
	for _, want := range []string{path, "permission denied", "Repository chat will fail"} {
		if !strings.Contains(got, want) {
			t.Errorf("warning %q does not contain %q", got, want)
		}
	}
}
