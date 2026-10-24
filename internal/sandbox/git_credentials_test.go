package sandbox

import (
	"os"
	"regexp"
	"testing"
)

// Both terminal credential paths must leave Git without a persistent token
// store. A future setup edit must not restore the old store helper.
func TestGitCredentialSetupDoesNotStoreToken(t *testing.T) {
	script, err := os.ReadFile(setupCredentialsScript)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{
		"sandbox create and provision": buildGitCredentialScript("example-token"),
		"credential setup script":      string(script),
	}
	store := regexp.MustCompile(`credential\.helper\s+["']?store\b`)
	write := regexp.MustCompile(`(?m)(?:>|tee\s+)\s*~?/?.*\.git-credentials`)
	for name, source := range paths {
		t.Run(name, func(t *testing.T) {
			if store.MatchString(source) {
				t.Error("Git credential store is configured")
			}
			if write.MatchString(source) {
				t.Error("the setup writes a Git credential file")
			}
		})
	}
}
