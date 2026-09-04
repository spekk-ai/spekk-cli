package sandbox

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	// awsTemplateFile is the CloudFormation template that stands up a
	// sandbox on AWS. It lives beside cloud-init.yaml because its UserData
	// is a copy of it.
	awsTemplateFile = "aws-cloudformation.yaml"

	// awsUserDataHeader is the line that opens the UserData block scalar in
	// the template. The block that follows it is the cloud-init content.
	awsUserDataHeader = "Fn::Base64: !Sub |"

	// awsAgentKeyVariable is the Fn::Sub variable that fills the agent
	// user's public key placeholder on AWS. On a droplet spekk fills the
	// same line with a key it generated.
	awsAgentKeyVariable = "${AgentPublicKey}"
)

// The template's UserData has to give an instance the same preparation that
// cloud-init.yaml gives a droplet, because spekk registers the instance as a
// machine it did not create and then deploys onto it without any preparation
// of its own. The two files are a copy, and a copy drifts unless something
// fails when it does.
func TestAWSUserDataMatchesCloudInit(t *testing.T) {
	template, err := os.ReadFile(awsTemplateFile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := blockScalarAfter(string(template), awsUserDataHeader)
	if err != nil {
		t.Fatal(err)
	}
	want := renderCloudInit(cloudInitTemplate, awsAgentKeyVariable)
	if got == want {
		return
	}
	line, gotLine, wantLine := firstDifference(got, want)
	t.Errorf("%s UserData differs from cloud-init.yaml at line %d:\n  template:   %q\n  cloud-init: %q\nEdit cloud-init.yaml first, then copy it into the template with the key placeholder mapped to %s.",
		awsTemplateFile, line, gotLine, wantLine, awsAgentKeyVariable)
}

// Fn::Sub reads every "${...}" in the UserData as a variable, and an unknown
// one fails the stack at create time, well after this repository's CI has
// passed. So the source file must not contain one: the only variable in the
// template is the one the test above maps the key placeholder to.
func TestCloudInitHasNoSubVariable(t *testing.T) {
	if i := strings.Index(string(cloudInitTemplate), "${"); i >= 0 {
		line := 1 + strings.Count(string(cloudInitTemplate[:i]), "\n")
		t.Errorf("cloud-init.yaml line %d contains \"${\", which Fn::Sub in %s reads as a variable; write it as \"${!\" in the template, or avoid it", line, awsTemplateFile)
	}
}

// blockScalarAfter returns the YAML literal block scalar that follows the
// one line whose content is header, with the block's indentation removed.
// A literal block with the default chomping keeps one trailing newline, and
// so does the result.
func blockScalarAfter(doc, header string) (string, error) {
	lines := strings.Split(doc, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) != header {
			continue
		}
		if start >= 0 {
			return "", fmt.Errorf("%q appears more than once", header)
		}
		start = i + 1
	}
	if start < 0 {
		return "", fmt.Errorf("%q not found", header)
	}

	indent := -1
	var block []string
	for _, l := range lines[start:] {
		if strings.TrimSpace(l) == "" {
			block = append(block, "")
			continue
		}
		lead := len(l) - len(strings.TrimLeft(l, " "))
		if indent < 0 {
			indent = lead
		}
		if lead < indent {
			break
		}
		block = append(block, l[indent:])
	}
	if indent < 0 {
		return "", fmt.Errorf("%q is followed by an empty block", header)
	}
	return strings.TrimRight(strings.Join(block, "\n"), "\n") + "\n", nil
}

// firstDifference returns the 1-based number of the first line on which a
// and b differ, and that line from each.
func firstDifference(a, b string) (int, string, string) {
	al := strings.Split(a, "\n")
	bl := strings.Split(b, "\n")
	for i := 0; i < len(al) || i < len(bl); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			return i + 1, x, y
		}
	}
	return 0, "", ""
}
