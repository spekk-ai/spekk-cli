package sandbox

import (
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
)

// spekkLinkRe finds the link that puts the spekk CLI on the system PATH.
var spekkLinkRe = regexp.MustCompile(`ln -sf (\S+) /usr/local/bin/spekk`)

// The link on the system PATH must point outside every home directory.
// Ubuntu creates a home 0750, so a link into agent's home works for agent
// and root only, and a non-root login such as `ubuntu` on AWS or `spekk` on
// Google Cloud gets "command not found". Both files that prepare a machine
// are checked, because they must stay in step.
func TestSpekkCLILinkWorksForEveryUser(t *testing.T) {
	prepare, err := os.ReadFile("../../scripts/prepare-machine.sh")
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"cloud-init.yaml":    string(cloudInitTemplate),
		"prepare-machine.sh": string(prepare),
	} {
		m := spekkLinkRe.FindStringSubmatch(content)
		if m == nil {
			t.Errorf("%s: no link puts spekk on the system PATH", name)
			continue
		}
		if strings.HasPrefix(m[1], "/home/") || strings.HasPrefix(m[1], "~") {
			t.Errorf("%s: /usr/local/bin/spekk points into a home directory (%s), which other users cannot enter", name, m[1])
		}
		// The install must put the binary where the link points, or the
		// link points at nothing.
		if want := "SPEKK_INSTALL_DIR=" + path.Dir(m[1]) + " sh"; !strings.Contains(content, want) {
			t.Errorf("%s: the link points to %s, but no install has %q", name, m[1], want)
		}
	}
}
