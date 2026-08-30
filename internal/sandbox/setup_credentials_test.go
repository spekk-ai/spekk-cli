package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupCredentialsScript provisions a droplet's credentials. Its env-file
// rendering is a pure function, so these tests source the script and call it
// directly rather than standing up a droplet.
var setupCredentialsScript = filepath.Join("..", "..", "infrastructure", "sandbox", "setup-credentials.sh")

// runScript runs a bash snippet with a minimal environment, so a credential in
// the developer's own shell cannot make a test pass.
func runScript(t *testing.T, env []string, snippet string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", "-c", snippet)
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH")}, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestSetupCredentialsRendersOneModesCredential(t *testing.T) {
	shared := []string{
		"SPEKK_AGENT_TOKEN=agenttok",
		"SPEKK_HOST=host.example",
		"GITHUB_TOKEN=ghtok",
		"AWS_ACCESS_KEY_ID=akid",
		"AWS_SECRET_ACCESS_KEY=secret",
		"AWS_DEFAULT_REGION=us-east-1",
		"CLAUDE_CODE_OAUTH_TOKEN=oauthtok",
	}

	tests := []struct {
		mode    string
		present []string
		absent  []string
	}{
		{
			mode:    "bedrock",
			present: []string{"AWS_ACCESS_KEY_ID=akid", "CLAUDE_CODE_USE_BEDROCK=1"},
			absent:  []string{"CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY"},
		},
		{
			mode:    "subscription",
			present: []string{"CLAUDE_CODE_OAUTH_TOKEN=oauthtok"},
			absent: []string{
				"CLAUDE_CODE_USE_BEDROCK",
				"ANTHROPIC_API_KEY",
				"AWS_ACCESS_KEY_ID",
				"AWS_SECRET_ACCESS_KEY",
				"AWS_DEFAULT_REGION",
			},
		},
	}

	for _, tt := range tests {
		env := append([]string{"SPEKK_AUTH_MODE=" + tt.mode}, shared...)
		got, err := runScript(t, env, "source "+setupCredentialsScript+"; render_agent_env")
		if err != nil {
			t.Fatalf("%s mode: rendering failed: %s\n%s", tt.mode, err, got)
		}

		for _, want := range []string{"GITHUB_TOKEN=ghtok", "SPEKK_HOST=host.example", "SPEKK_AGENT_TOKEN=agenttok"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s mode: missing shared line %q", tt.mode, want)
			}
		}
		for _, want := range tt.present {
			if !strings.Contains(got, want) {
				t.Errorf("%s mode: missing %q", tt.mode, want)
			}
		}
		for _, unwanted := range tt.absent {
			if strings.Contains(got, unwanted) {
				t.Errorf("%s mode: file still carries %q:\n%s", tt.mode, unwanted, got)
			}
		}
	}
}

// A switch is only a switch if the previous mode's variables are gone. The file
// is rewritten whole, so seeding it with a Bedrock block and rendering the
// subscription mode over it must leave nothing behind.
func TestSetupCredentialsSwitchDropsThePreviousMode(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "agent.env")
	seeded := "AWS_ACCESS_KEY_ID=old\nCLAUDE_CODE_USE_BEDROCK=1\nGITHUB_TOKEN=old\n"
	if err := os.WriteFile(envFile, []byte(seeded), 0o600); err != nil {
		t.Fatalf("seeding env file: %s", err)
	}

	env := []string{
		"SPEKK_AUTH_MODE=subscription",
		"CLAUDE_CODE_OAUTH_TOKEN=newtok",
		"GITHUB_TOKEN=ghtok",
		"SPEKK_HOST=host.example",
		"SPEKK_AGENT_TOKEN=agenttok",
	}
	out, err := runScript(t, env, "source "+setupCredentialsScript+"; render_agent_env > "+envFile)
	if err != nil {
		t.Fatalf("rendering failed: %s\n%s", err, out)
	}

	got, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %s", err)
	}
	if !strings.Contains(string(got), "CLAUDE_CODE_OAUTH_TOKEN=newtok") {
		t.Errorf("switched file is missing the subscription token:\n%s", got)
	}
	for _, unwanted := range []string{"AWS_ACCESS_KEY_ID", "CLAUDE_CODE_USE_BEDROCK"} {
		if strings.Contains(string(got), unwanted) {
			t.Errorf("switched file still carries %q:\n%s", unwanted, got)
		}
	}
}

func TestSetupCredentialsRejectsUnknownMode(t *testing.T) {
	out, err := runScript(t, []string{"SPEKK_AUTH_MODE=bedrok"}, "source "+setupCredentialsScript)
	if err == nil {
		t.Fatal("an unknown auth mode was accepted")
	}
	for _, want := range []string{"bedrok", "bedrock", "subscription"} {
		if !strings.Contains(out, want) {
			t.Errorf("rejection message %q does not mention %q", strings.TrimSpace(out), want)
		}
	}
}

// Every credential must be suppliable from the environment, or an unattended
// run blocks on a prompt nobody can answer. stdin is closed, so a prompt that
// fires anyway fails the read and, under `set -e`, the whole script.
func TestSetupCredentialsPromptsNothingWhenTheEnvironmentIsComplete(t *testing.T) {
	for _, mode := range []string{"bedrock", "subscription"} {
		env := []string{
			"SPEKK_AUTH_MODE=" + mode,
			"SPEKK_AGENT_TOKEN=agenttok",
			"SPEKK_HOST=host.example",
			"GITHUB_TOKEN=ghtok",
			"AWS_ACCESS_KEY_ID=akid",
			"AWS_SECRET_ACCESS_KEY=secret",
			"AWS_DEFAULT_REGION=us-east-1",
			"CLAUDE_CODE_OAUTH_TOKEN=oauthtok",
		}
		out, err := runScript(t, env, "source "+setupCredentialsScript+"; collect_credentials < /dev/null")
		if err != nil {
			t.Errorf("%s mode: a value already in the environment was prompted for anyway: %s\n%s", mode, err, out)
		}
	}
}

// A switch rewrites the env file whole, which is what makes it a switch. That
// same property would discard every value the new mode does not decide, so the
// script reads the file before replacing it. The agent token is the one that
// cannot be recovered: the control host stores only its hash, so losing it
// means re-registering the agent.
func TestSetupCredentialsSwitchCarriesForwardWhatItDoesNotDecide(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "agent.env")
	seeded := "AWS_ACCESS_KEY_ID=AKIAOLD\n" +
		"AWS_SECRET_ACCESS_KEY=oldsecret\n" +
		"AWS_DEFAULT_REGION=us-east-1\n" +
		"CLAUDE_CODE_USE_BEDROCK=1\n" +
		"GITHUB_TOKEN=ghp_existing\n" +
		"SPEKK_HOST=app.example\n" +
		"SPEKK_AGENT_TOKEN=agenttoken-that-must-survive\n" +
		"WORKSPACE=/opt/spekk/workspace\n" +
		"SPEKK_AGENT_NAME=spekk-box\n"
	if err := os.WriteFile(envFile, []byte(seeded), 0o600); err != nil {
		t.Fatalf("seeding env file: %s", err)
	}

	// Only the new mode's own credential is supplied, as an operator running
	// the switch would supply it.
	env := []string{
		"SPEKK_AUTH_MODE=subscription",
		"CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat-new",
		"AGENT_ENV_FILE=" + envFile,
	}
	// Drive the whole write path, not just the pieces, so a version that
	// forgets to read the file before replacing it fails here.
	if out, err := runScript(t, env, "source "+setupCredentialsScript+"; write_agent_env"); err != nil {
		t.Fatalf("write failed: %s\n%s", err, out)
	}
	written, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %s", err)
	}
	out := string(written)

	for _, want := range []string{
		"CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat-new",
		"SPEKK_AGENT_TOKEN=agenttoken-that-must-survive",
		"GITHUB_TOKEN=ghp_existing",
		"SPEKK_HOST=app.example",
		"WORKSPACE=/opt/spekk/workspace",
		"SPEKK_AGENT_NAME=spekk-box",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("switch lost %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "CLAUDE_CODE_USE_BEDROCK", "ANTHROPIC_API_KEY"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("switch kept %q from the old mode:\n%s", unwanted, out)
		}
	}
}

// An empty answer at a credential prompt must not become an empty credential.
func TestSetupCredentialsRejectsAnEmptySecret(t *testing.T) {
	out, err := runScript(t, nil,
		"source "+setupCredentialsScript+"; printf '\\n\\nvalue\\n' | { prompt_secret TESTVAR 'token'; echo \"got=$TESTVAR\"; }")
	if err != nil {
		t.Fatalf("prompt failed: %s\n%s", err, out)
	}
	if !strings.Contains(out, "got=value") {
		t.Errorf("prompt should have kept asking until a value arrived:\n%s", out)
	}
}

// A real sandbox turned out to carry ANTHROPIC_MODEL, which no list in the
// script anticipated. Carrying forward by name would have dropped it and
// quietly changed which model the agent runs, so the script keeps everything
// that is not the mode's own credential, known to it or not.
func TestSetupCredentialsCarriesForwardUnknownVariables(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "agent.env")
	seeded := "AWS_ACCESS_KEY_ID=AKIAOLD\n" +
		"CLAUDE_CODE_USE_BEDROCK=1\n" +
		"GITHUB_TOKEN=ghp_existing\n" +
		"SPEKK_HOST=app.example\n" +
		"SPEKK_AGENT_TOKEN=agenttok\n" +
		"ANTHROPIC_MODEL=us.anthropic.claude-sonnet-5\n" +
		"SOME_FUTURE_SETTING=keepme\n"
	if err := os.WriteFile(envFile, []byte(seeded), 0o600); err != nil {
		t.Fatalf("seeding env file: %s", err)
	}

	env := []string{
		"SPEKK_AUTH_MODE=subscription",
		"CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat-new",
		"AGENT_ENV_FILE=" + envFile,
	}
	if out, err := runScript(t, env, "source "+setupCredentialsScript+"; write_agent_env"); err != nil {
		t.Fatalf("write failed: %s\n%s", err, out)
	}
	written, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %s", err)
	}
	got := string(written)

	if !strings.Contains(got, "SOME_FUTURE_SETTING=keepme") {
		t.Errorf("switch dropped a setting the mode does not decide:\n%s", got)
	}
	// ANTHROPIC_MODEL is not a credential, but its value is mode-specific: a
	// Bedrock inference profile is not a model the subscription API knows, and
	// carrying it over makes every turn fail on the selected model.
	for _, unwanted := range []string{"ANTHROPIC_MODEL", "AWS_ACCESS_KEY_ID", "CLAUDE_CODE_USE_BEDROCK"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("switch kept %q from the old mode:\n%s", unwanted, got)
		}
	}
}

// A model pin belongs to the mode whose API knows the name, so a switch cannot
// carry one over. It can still honor a pin the operator supplies for the mode
// they are switching to, which is what lets them move it in one step.
func TestSetupCredentialsWritesAnOperatorSuppliedModelPin(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "agent.env")
	seeded := "CLAUDE_CODE_USE_BEDROCK=1\n" +
		"ANTHROPIC_MODEL=us.anthropic.claude-sonnet-5\n" +
		"GITHUB_TOKEN=gh\nSPEKK_HOST=h\nSPEKK_AGENT_TOKEN=t\n"
	if err := os.WriteFile(envFile, []byte(seeded), 0o600); err != nil {
		t.Fatalf("seeding env file: %s", err)
	}

	base := []string{
		"SPEKK_AUTH_MODE=subscription",
		"CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat-new",
		"AGENT_ENV_FILE=" + envFile,
	}

	// Without a pin from the operator, the old one is gone.
	if out, err := runScript(t, base, "source "+setupCredentialsScript+"; load_existing; render_agent_env"); err != nil {
		t.Fatalf("render failed: %s\n%s", err, out)
	} else if strings.Contains(out, "ANTHROPIC_MODEL") {
		t.Errorf("the previous mode's pin must not survive:\n%s", out)
	}

	// With one, it is written.
	withPin := append(append([]string{}, base...), "ANTHROPIC_MODEL=claude-sonnet-5")
	out, err := runScript(t, withPin, "source "+setupCredentialsScript+"; load_existing; render_agent_env")
	if err != nil {
		t.Fatalf("render failed: %s\n%s", err, out)
	}
	if !strings.Contains(out, "ANTHROPIC_MODEL=claude-sonnet-5") {
		t.Errorf("an operator-supplied pin should be written:\n%s", out)
	}
	if strings.Contains(out, "us.anthropic") {
		t.Errorf("the old pin leaked through:\n%s", out)
	}
}

// Dropping a pin without saying so leaves a sandbox that answers differently
// tomorrow for no visible reason.
func TestSetupCredentialsReportsADroppedModelPin(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "agent.env")
	if err := os.WriteFile(envFile, []byte("ANTHROPIC_MODEL=us.anthropic.claude-sonnet-5\n"), 0o600); err != nil {
		t.Fatalf("seeding env file: %s", err)
	}
	out, err := runScript(t, []string{"SPEKK_AUTH_MODE=subscription", "AGENT_ENV_FILE=" + envFile},
		"source "+setupCredentialsScript+"; load_existing; echo \"dropped=$DROPPED_MODEL\"")
	if err != nil {
		t.Fatalf("load failed: %s\n%s", err, out)
	}
	if !strings.Contains(out, "dropped=us.anthropic.claude-sonnet-5") {
		t.Errorf("the dropped pin should be remembered so it can be reported:\n%s", out)
	}
}
