package sandbox

import (
	"strings"
	"testing"
)

func TestParseAuthMode(t *testing.T) {
	tests := []struct {
		input string
		want  AuthMode
	}{
		{"", AuthBedrock}, // no --auth flag keeps the original behavior
		{"bedrock", AuthBedrock},
		{"subscription", AuthSubscription},
	}
	for _, tt := range tests {
		got, err := ParseAuthMode(tt.input)
		if err != nil {
			t.Errorf("ParseAuthMode(%q) returned error: %s", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseAuthMode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseAuthModeRejectsUnknownMode(t *testing.T) {
	_, err := ParseAuthMode("bedrok")
	if err == nil {
		t.Fatal("ParseAuthMode(\"bedrok\") returned no error")
	}
	// The message has to tell the operator what to type next, so it names both
	// the value it got and the two it accepts.
	for _, want := range []string{"bedrok", "bedrock", "subscription"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestRequiredEnvVars(t *testing.T) {
	bedrock := requiredEnvVars(AuthBedrock)
	for _, want := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN", "SPEKK_HOST"} {
		if !contains(bedrock, want) {
			t.Errorf("bedrock mode does not require %s", want)
		}
	}

	subscription := requiredEnvVars(AuthSubscription)
	for _, want := range []string{oauthTokenVar, "GITHUB_TOKEN", "SPEKK_HOST"} {
		if !contains(subscription, want) {
			t.Errorf("subscription mode does not require %s", want)
		}
	}
	// A subscription sandbox never calls AWS, so absent AWS credentials must not
	// block it.
	for _, v := range subscription {
		if strings.HasPrefix(v, "AWS_") {
			t.Errorf("subscription mode requires %s, which it never uses", v)
		}
	}
}

// testEnvVars is a full set of credentials, so a mode that leaks the other
// mode's variables leaks a non-empty value the test can see.
func testEnvVars() map[string]string {
	return map[string]string{
		"AWS_ACCESS_KEY_ID":     "akid",
		"AWS_SECRET_ACCESS_KEY": "secret",
		"AWS_DEFAULT_REGION":    "us-east-1",
		"GITHUB_TOKEN":          "ghtok",
		oauthTokenVar:           "oauthtok",
	}
}

func TestBuildEnvContentBedrockPinsTheDefaultFile(t *testing.T) {
	got := buildEnvContent(AuthBedrock, testEnvVars(), "probe", "agenttok", "agent-1", "https://host.example/")

	want := strings.Join([]string{
		"AWS_ACCESS_KEY_ID=akid",
		"AWS_SECRET_ACCESS_KEY=secret",
		"AWS_DEFAULT_REGION=us-east-1",
		"CLAUDE_CODE_USE_BEDROCK=1",
		"GITHUB_TOKEN=ghtok",
		"SPEKK_HOST=host.example",
		"SPEKK_AGENT_TOKEN=agenttok",
		"SPEKK_AGENT_ID=agent-1",
		"WORKSPACE=/opt/spekk/workspace",
		"SPEKK_AGENT_NAME=spekk-probe",
	}, "\n") + "\n"

	if got != want {
		t.Errorf("bedrock env file changed:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// The bedrock file is pinned above, which already proves what it does and does
// not contain. This covers the other direction: subscription mode must carry
// none of the credentials the pinned file carries. A file that sets the
// subscription token and leaves CLAUDE_CODE_USE_BEDROCK behind can keep billing
// through Bedrock, which is the failure the auth mode exists to prevent.
func TestBuildEnvContentSubscriptionCarriesNoBedrockCredential(t *testing.T) {
	got := buildEnvContent(AuthSubscription, testEnvVars(), "probe", "agenttok", "agent-1", "https://host.example/")

	if !strings.Contains(got, oauthTokenVar+"=oauthtok") {
		t.Errorf("subscription file is missing the token:\n%s", got)
	}
	// Shared lines survive whichever credential was chosen.
	for _, want := range []string{"GITHUB_TOKEN=ghtok", "SPEKK_HOST=host.example", "SPEKK_AGENT_TOKEN=agenttok"} {
		if !strings.Contains(got, want) {
			t.Errorf("subscription file is missing shared line %q", want)
		}
	}
	for _, unwanted := range []string{
		"CLAUDE_CODE_USE_BEDROCK",
		"ANTHROPIC_API_KEY",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_DEFAULT_REGION",
	} {
		if strings.Contains(got, unwanted) {
			t.Errorf("subscription file still carries %q:\n%s", unwanted, got)
		}
	}
}

// A machine-mode file is pinned whole for the two types that the terminal
// path does not have, so a line of another type, or a GITHUB_TOKEN line with
// a token that expires in 1 hour, fails here. The bedrock block is the one
// the pinned bedrock file above already proves.
func TestBuildMachineEnvContentPinsEachNewType(t *testing.T) {
	shared := "SPEKK_HOST=host.example\nSPEKK_AGENT_TOKEN=agenttok\nSPEKK_AGENT_ID=agent-1\nWORKSPACE=/opt/spekk/workspace\nSPEKK_AGENT_NAME=spekk-probe\n"
	for _, tc := range []struct {
		auth AuthMode
		env  map[string]string
		want string
	}{
		{AuthNone, map[string]string{}, shared},
		{AuthAPIKey, map[string]string{apiKeyVar: "sk-key"}, "ANTHROPIC_API_KEY=sk-key\n" + shared},
	} {
		in := &machineInput{auth: tc.auth, modelEnv: tc.env, controlHost: "https://host.example", githubToken: "ghtok"}
		if got := buildMachineEnvContent(in, "probe", "agenttok", "agent-1"); got != tc.want {
			t.Errorf("%s env file:\ngot:\n%s\nwant:\n%s", tc.auth, got, tc.want)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
