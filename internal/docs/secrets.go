package docs

import (
	"regexp"
	"strings"
)

// SecretHit is one secret found in a document: what kind it is and its line.
// It never holds the secret itself, so no message can print it.
type SecretHit struct {
	Kind string
	Line int
}

var secretPatterns = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"private key", regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)},
	{"AWS access key id", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"GitHub token", regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{22,})\b`)},
	{"Slack token", regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`)},
	{"Anthropic API key", regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}`)},
	{"Google API key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
	{"PGP private key", regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`)},
	{"OpenAI API key", regexp.MustCompile(`\bsk-(?:(?:proj|svcacct|admin)-[A-Za-z0-9_-]{20,}|[A-Za-z0-9]{32,})`)},
	{"Stripe live key", regexp.MustCompile(`\b[rs]k_live_[0-9A-Za-z]{16,}`)},
	{"GitLab token", regexp.MustCompile(`\bglpat-[0-9A-Za-z_-]{20,}`)},
	// A .env line: an upper-case name that says what it holds, and a value
	// that is not a placeholder ($VAR, <value>, {{value}}).
	{"password or secret assignment", regexp.MustCompile(`^\s*(?:export\s+)?[A-Z0-9_]*(?:PASSWORD|SECRET|TOKEN|API_KEY)[A-Z0-9_]*\s*=\s*["']?[^\s"'$<{]{8,}`)},
}

// ScanSecrets returns each line of text that holds a secret, one hit for each
// kind on a line, in line order.
func ScanSecrets(text string) []SecretHit {
	var hits []SecretHit
	for i, line := range strings.Split(text, "\n") {
		for _, p := range secretPatterns {
			if p.re.MatchString(line) {
				hits = append(hits, SecretHit{Kind: p.kind, Line: i + 1})
			}
		}
	}
	return hits
}
