package sandbox

import "fmt"

// AuthMode names the credential a sandbox authenticates Claude with.
type AuthMode string

const (
	// AuthBedrock bills Claude usage through the AWS Bedrock API.
	AuthBedrock AuthMode = "bedrock"
	// AuthSubscription authenticates with a long-lived Claude subscription
	// token, minted by `claude setup-token`.
	AuthSubscription AuthMode = "subscription"

	// AuthAPIKey and AuthNone are for machine mode only, so ParseAuthMode
	// does not accept them. AuthAPIKey bills through an Anthropic API key.
	// AuthNone writes no model credential: the person signs in to Claude
	// Code on the machine later.
	AuthAPIKey AuthMode = "api-key"
	AuthNone   AuthMode = "none"
)

// apiKeyVar is the variable Claude Code reads an Anthropic API key from.
const apiKeyVar = "ANTHROPIC_API_KEY"

// oauthTokenVar is the variable Claude Code reads a subscription token from.
const oauthTokenVar = "CLAUDE_CODE_OAUTH_TOKEN"

// codexAuthFileVar points to a local, file-backed ChatGPT login cache.
// The cache itself is sent over SSH stdin and never placed in an argument.
const codexAuthFileVar = "CODEX_AUTH_JSON_FILE"

// ParseAuthMode maps the --auth flag's text to a mode. An empty string is
// bedrock, so an operator who passes no flag keeps the behavior the command
// had before the flag existed.
func ParseAuthMode(s string) (AuthMode, error) {
	switch AuthMode(s) {
	case "", AuthBedrock:
		return AuthBedrock, nil
	case AuthSubscription:
		return AuthSubscription, nil
	}
	return "", fmt.Errorf("invalid auth mode %q: must be %q or %q", s, AuthBedrock, AuthSubscription)
}

// requiredEnvVars lists the variables Create needs before it creates anything
// billable. GITHUB_TOKEN and SPEKK_HOST are needed whatever the sandbox pays
// with; the rest follow the credential, so a subscription sandbox is not
// blocked by AWS credentials it will never use.
func requiredEnvVars(mode AuthMode) []string {
	if mode == AuthSubscription {
		return []string{oauthTokenVar, "GITHUB_TOKEN", "SPEKK_HOST"}
	}
	return []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN", "SPEKK_HOST"}
}

// authLines returns the model-credential lines for a mode.
//
// Only the chosen mode's variables appear. CLAUDE_CODE_USE_BEDROCK and
// CLAUDE_CODE_OAUTH_TOKEN each select a credential path of their own, so a file
// carrying both would leave the choice to whichever one Claude Code happens to
// read first — and a sandbox switched to a subscription could keep billing
// through Bedrock with nothing to show for it.
func authLines(mode AuthMode, envVars map[string]string) []string {
	switch mode {
	case AuthSubscription:
		return []string{oauthTokenVar + "=" + envVars[oauthTokenVar]}
	case AuthAPIKey:
		return []string{apiKeyVar + "=" + envVars[apiKeyVar]}
	case AuthNone:
		return nil
	}
	return []string{
		"AWS_ACCESS_KEY_ID=" + envVars["AWS_ACCESS_KEY_ID"],
		"AWS_SECRET_ACCESS_KEY=" + envVars["AWS_SECRET_ACCESS_KEY"],
		"AWS_DEFAULT_REGION=" + envVars["AWS_DEFAULT_REGION"],
		"CLAUDE_CODE_USE_BEDROCK=1",
	}
}
