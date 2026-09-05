package sandbox

import (
	"fmt"
	"strings"
)

// The CloudFormation template in this directory stands up a sandbox on AWS.
// Its UserData is the cloud-init content from cloud-init.yaml, rendered by
// AWSUserData, and `go generate` writes that block into the template so the
// copy is never made by hand. TestAWSUserDataMatchesCloudInit fails when
// the two drift.
//
//go:generate go run ./awsuserdata

const (
	// AWSTemplateFile is the CloudFormation template that stands up a
	// sandbox on AWS. It lives beside cloud-init.yaml because its UserData
	// is a copy of it.
	AWSTemplateFile = "aws-cloudformation.yaml"

	// AWSUserDataHeader is the line that opens the UserData block scalar in
	// the template. The block that follows it is the cloud-init content.
	AWSUserDataHeader = "Fn::Base64: !Sub |"

	// awsAgentKeyVariable is the Fn::Sub variable that fills the agent
	// user's public key placeholder on AWS. On a droplet spekk fills the
	// same line with a key it generated.
	awsAgentKeyVariable = "${AgentPublicKey}"

	// cloudInitUsersKey opens the users list in cloud-init.yaml.
	cloudInitUsersKey = "users:\n"

	// awsDefaultUserEntry is the entry the template adds at the top of that
	// list. A users list without it makes cloud-init create the listed
	// users only, so the image's ubuntu user, the login KeyName is for,
	// never exists. A droplet logs in as root and does not need it.
	awsDefaultUserEntry = "  - default\n"
)

// AWSUserData returns the UserData the template must carry for the embedded
// cloud-init.yaml: the key placeholder mapped to the Fn::Sub variable, and
// the default user entry at the top of the users list.
func AWSUserData() string {
	rendered := renderCloudInit(cloudInitTemplate, awsAgentKeyVariable)
	return strings.Replace(rendered, cloudInitUsersKey, cloudInitUsersKey+awsDefaultUserEntry, 1)
}

// BlockScalarAfter returns the YAML literal block scalar that follows the
// one line whose content is header, with the block's indentation removed.
// A literal block with the default chomping keeps one trailing newline, and
// so does the result.
func BlockScalarAfter(doc, header string) (string, error) {
	lines := strings.Split(doc, "\n")
	start, end, indent, err := blockScalarBounds(lines, header)
	if err != nil {
		return "", err
	}
	block := make([]string, 0, end-start)
	for _, l := range lines[start:end] {
		if strings.TrimSpace(l) == "" {
			block = append(block, "")
			continue
		}
		block = append(block, l[indent:])
	}
	return strings.TrimRight(strings.Join(block, "\n"), "\n") + "\n", nil
}

// ReplaceBlockScalarAfter returns doc with the block scalar that follows
// header replaced by block, indented as the old block was. One empty line
// separates the new block from the key that follows it, as in the source
// this repository keeps.
func ReplaceBlockScalarAfter(doc, header, block string) (string, error) {
	lines := strings.Split(doc, "\n")
	start, end, indent, err := blockScalarBounds(lines, header)
	if err != nil {
		return "", err
	}
	pad := strings.Repeat(" ", indent)
	var out []string
	out = append(out, lines[:start]...)
	for _, l := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
		if l == "" {
			out = append(out, "")
			continue
		}
		out = append(out, pad+l)
	}
	out = append(out, "")
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n"), nil
}

// blockScalarBounds finds the block scalar that follows the one line whose
// content is header: the index of its first line, the index of the first
// line after it, and its indentation. The block ends at the first non-empty
// line indented less than its first line.
func blockScalarBounds(lines []string, header string) (start, end, indent int, err error) {
	start = -1
	for i, l := range lines {
		if strings.TrimSpace(l) != header {
			continue
		}
		if start >= 0 {
			return 0, 0, 0, fmt.Errorf("%q appears more than once", header)
		}
		start = i + 1
	}
	if start < 0 {
		return 0, 0, 0, fmt.Errorf("%q not found", header)
	}

	indent = -1
	end = len(lines)
	for i := start; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		lead := len(l) - len(strings.TrimLeft(l, " "))
		if indent < 0 {
			indent = lead
			continue
		}
		if lead < indent {
			end = i
			break
		}
	}
	if indent < 0 {
		return 0, 0, 0, fmt.Errorf("%q is followed by an empty block", header)
	}
	return start, end, indent, nil
}
