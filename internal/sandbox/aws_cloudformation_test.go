package sandbox

import (
	"os"
	"strings"
	"testing"
)

// The template's UserData has to give an instance the same preparation that
// cloud-init.yaml gives a droplet, because spekk registers the instance as a
// machine it did not create and then deploys onto it without any preparation
// of its own. The two files are a copy with two declared differences, the
// key variable and the default user entry, and a copy drifts unless
// something fails when it does.
func TestAWSUserDataMatchesCloudInit(t *testing.T) {
	template, err := os.ReadFile(AWSTemplateFile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := BlockScalarAfter(string(template), AWSUserDataHeader)
	if err != nil {
		t.Fatal(err)
	}
	want := AWSUserData()
	if got == want {
		return
	}
	line, gotLine, wantLine := firstDifference(got, want)
	t.Errorf("%s UserData differs from cloud-init.yaml at line %d:\n  template:   %q\n  cloud-init: %q\nRun \"go generate ./internal/sandbox\" to write the block from cloud-init.yaml.",
		AWSTemplateFile, line, gotLine, wantLine)
}

// The generator must leave a template that is already current alone. A
// generator that rewrites an unchanged block, even by one blank line, makes
// `go generate` and the drift test disagree.
func TestAWSUserDataGeneratorIsStable(t *testing.T) {
	template, err := os.ReadFile(AWSTemplateFile)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := ReplaceBlockScalarAfter(string(template), AWSUserDataHeader, AWSUserData())
	if err != nil {
		t.Fatal(err)
	}
	if updated != string(template) {
		line, gotLine, wantLine := firstDifference(updated, string(template))
		t.Errorf("regenerating %s changes it at line %d:\n  generated: %q\n  on disk:   %q", AWSTemplateFile, line, gotLine, wantLine)
	}
}

// Fn::Sub reads every "${...}" in the UserData as a variable, and an unknown
// one fails the stack at create time, well after this repository's CI has
// passed. So the source file must not contain one: the only variable in the
// template is the one AWSUserData maps the key placeholder to.
func TestCloudInitHasNoSubVariable(t *testing.T) {
	if i := strings.Index(string(cloudInitTemplate), "${"); i >= 0 {
		line := 1 + strings.Count(string(cloudInitTemplate[:i]), "\n")
		t.Errorf("cloud-init.yaml line %d contains \"${\", which Fn::Sub in %s reads as a variable; write it as \"${!\" in the template, or avoid it", line, AWSTemplateFile)
	}
}

// The default user entry is one line in one place. If cloud-init.yaml ever
// has no users list, or more than one, AWSUserData is wrong and the drift
// test would pass on a template that is wrong in the same way.
func TestCloudInitHasOneUsersList(t *testing.T) {
	if n := strings.Count(string(cloudInitTemplate), cloudInitUsersKey); n != 1 {
		t.Errorf("cloud-init.yaml has %d %q lines, want 1", n, strings.TrimSpace(cloudInitUsersKey))
	}
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
