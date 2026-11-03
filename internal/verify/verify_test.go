package verify

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

func write(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "config", "user.email", "test@example.test")
	git(t, root, "config", "user.name", "Test")
	write(t, filepath.Join(root, ".gitignore"), ".spekk/index.db\n.spekk/index.db-*\n")
	write(t, filepath.Join(root, "go.mod"), "module example.test/acme\n\ngo 1.25\n")
	write(t, filepath.Join(root, "specs", "sample", "sample.md"), "---\nid: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Sample\n")
	return root
}
func propertyFile(t *testing.T, root, id, extra string) {
	t.Helper()
	write(t, filepath.Join(root, "specs", "sample", "assertions", id+".md"), fmt.Sprintf("---\nid: %s\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n%s---\n# %s\n", id, extra, id))
}
func commit(t *testing.T, root, message string) string {
	t.Helper()
	git(t, root, "add", ".")
	git(t, root, "commit", "-q", "-m", message)
	return git(t, root, "rev-parse", "HEAD")
}
func config(t *testing.T, root, command, glob string) {
	t.Helper()
	write(t, filepath.Join(root, "spekk.toml"), fmt.Sprintf("[[verify.run]]\ncommand = %q\njunit = %q\n", command, glob))
}
func run(t *testing.T, root string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := Run(filepath.Join(root, "specs"), nil, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}
func db(t *testing.T, root string) *sql.DB {
	t.Helper()
	d, err := sql.Open("sqlite", index.DBPath(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
func truth(t *testing.T, d *sql.DB, id string) (string, string) {
	t.Helper()
	var tr, level string
	err := d.QueryRow(`SELECT truth, level FROM verify_truth WHERE assertion_id = ? ORDER BY run_id DESC LIMIT 1`, id).Scan(&tr, &level)
	if err != nil {
		t.Fatal(err)
	}
	return tr, level
}

func TestVerifyRunsConfiguredCommandsAndRejectsStaleFiles(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "")
	write(t, filepath.Join(root, "checks_test.go"), "// spekk: check\nfunc TestCheck(t *testing.T) {}\n")
	commit(t, root, "Add project")
	xml := `<testsuite><testcase classname="example.test/acme" name="TestCheck"/></testsuite>`
	write(t, filepath.Join(root, "source.xml"), xml)
	config(t, root, "cp source.xml first.xml; echo first", "first.xml")
	f, err := os.OpenFile(filepath.Join(root, "spekk.toml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(f, "\n[[verify.run]]\ncommand = \"cp source.xml second.xml; echo second; exit 2\"\njunit = \"second.xml\"\n")
	f.Close()
	out, diagnostic, err := run(t, root)
	if err == nil || !strings.Contains(diagnostic, "first\nsecond\n") || !strings.Contains(out, "1 verified") {
		t.Fatalf("out=%q stderr=%q err=%v", out, diagnostic, err)
	}
	if got, _ := truth(t, db(t, root), "check"); got != "verified" {
		t.Fatal(got)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(root, "first.xml"), old, old); err != nil {
		t.Fatal(err)
	}
	config(t, root, "true", "first.xml")
	_, diagnostic, err = run(t, root)
	if err == nil || !strings.Contains(diagnostic, "not written by this run") {
		t.Fatalf("stale: %q %v", diagnostic, err)
	}
	config(t, root, "true", "absent*.xml")
	_, diagnostic, err = run(t, root)
	if err == nil || !strings.Contains(diagnostic, "not written by this run") {
		t.Fatalf("missing: %q %v", diagnostic, err)
	}
	if err := os.Remove(filepath.Join(root, "spekk.toml")); err != nil {
		t.Fatal(err)
	}
	_, _, err = run(t, root)
	if err == nil || !strings.Contains(err.Error(), "configure") {
		t.Fatal(err)
	}
	var help bytes.Buffer
	if err := Run(filepath.Join(root, "specs"), []string{"--help"}, &help, &bytes.Buffer{}); err != nil || !strings.Contains(help.String(), "[[verify.run]]") {
		t.Fatal(err, help.String())
	}
}

func TestVerifyAcceptsFreshJUnitFromShellBuiltin(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "")
	write(t, filepath.Join(root, "check_test.go"), "// spekk: check\nfunc TestCheck(t *testing.T) {}\n")
	commit(t, root, "Add check")
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestCheck"/></testsuite>`)
	config(t, root, `read -r line < source.xml; echo "$line" > result.xml`, "result.xml")
	for i := 0; i < 50; i++ {
		out, diagnostic, err := run(t, root)
		if err != nil || !strings.Contains(out, "1 verified") {
			t.Fatalf("run %d: stdout=%q stderr=%q err=%v", i+1, out, diagnostic, err)
		}
	}
}

func TestResultsMapByPackageAndModule(t *testing.T) {
	root := project(t)
	for _, id := range []string{"one", "two", "py-one", "py-two"} {
		propertyFile(t, root, id, "")
	}
	write(t, filepath.Join(root, "one", "same_test.go"), "// spekk: one\nfunc TestSame(t *testing.T) {}\n")
	write(t, filepath.Join(root, "two", "same_test.go"), "// spekk: two\nfunc TestSame(t *testing.T) {}\n")
	write(t, filepath.Join(root, "one", "test_same.py"), "class TestGroup:\n    @spekk('py-one')\n    def test_same(self): pass\n")
	write(t, filepath.Join(root, "two", "test_same.py"), "@spekk('py-two')\ndef test_same(): pass\n")
	commit(t, root, "Add tests")
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme/one" name="TestSame/case"/><testcase classname="one.test_same.TestGroup" name="test_same[a-1]"/><testcase classname="example.test/acme/other" name="TestSame"><failure/></testcase></testsuite>`)
	config(t, root, "cp source.xml result.xml", "result.xml")
	out, diagnostic, err := run(t, root)
	if err != nil {
		t.Fatal(err, out, diagnostic)
	}
	d := db(t, root)
	for _, id := range []string{"one", "py-one"} {
		if got, _ := truth(t, d, id); got != "verified" {
			t.Fatal(id, got)
		}
	}
	for _, id := range []string{"two", "py-two"} {
		if got, _ := truth(t, d, id); got != "unchecked" {
			t.Fatal(id, got)
		}
	}
	if !strings.Contains(diagnostic, "no result: two/same_test.go:TestSame") || !strings.Contains(diagnostic, "no result: two/test_same.py:test_same") {
		t.Fatal(diagnostic)
	}
}

func TestLevelAndFailurePriority(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "truth: verified\nlevel: proof\n")
	write(t, filepath.Join(root, "a_test.go"), "// spekk: check evidence=example\nfunc TestExample(t *testing.T) {}\n// spekk: check evidence=property\nfunc TestProperty(t *testing.T) {}\n")
	commit(t, root, "Add checks")
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestExample"/><testcase classname="example.test/acme" name="TestProperty"><failure/></testcase></testsuite>`)
	config(t, root, "cp source.xml result.xml", "result.xml")
	out, _, err := run(t, root)
	if err == nil || !strings.Contains(out, "check: failing: TestProperty") {
		t.Fatal(out, err)
	}
	if got, level := truth(t, db(t, root), "check"); got != "failing" || level != "example" {
		t.Fatal(got, level)
	}
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestExample"/><testcase classname="example.test/acme" name="TestProperty"/></testsuite>`)
	_, _, err = run(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if got, level := truth(t, db(t, root), "check"); got != "verified" || level != "property" {
		t.Fatal(got, level)
	}
}

func TestFailedNamesAreUniqueInFirstSeenOrder(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "")
	write(t, filepath.Join(root, "checks_test.go"), "// spekk: check\nfunc TestOne(t *testing.T) {}\n// spekk: check\nfunc TestTwo(t *testing.T) {}\n")
	commit(t, root, "Add checks")
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestTwo"><failure/></testcase><testcase classname="example.test/acme" name="TestOne"><failure/></testcase><testcase classname="example.test/acme" name="TestTwo/sub"><failure/></testcase></testsuite>`)
	config(t, root, "cp source.xml result.xml", "result.xml")
	out, _, err := run(t, root)
	if err == nil || !strings.Contains(out, "check: failing: TestTwo, TestOne\n") {
		t.Fatalf("unexpected failed names: %q, %v", out, err)
	}
}

func TestInvalidJUnitFileIsAnError(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "")
	commit(t, root, "Add property")
	for _, content := range []string{"", "<other><testcase name=\"TestCheck\"/></other>"} {
		write(t, filepath.Join(root, "source.xml"), content)
		config(t, root, "cp source.xml result.xml", "result.xml")
		_, diagnostic, err := run(t, root)
		if err == nil || !strings.Contains(diagnostic, "result.xml: not JUnit XML") {
			t.Fatalf("input %q: stderr %q, error %v", content, diagnostic, err)
		}
	}
	write(t, filepath.Join(root, "source.xml"), "<testsuite/>")
	_, diagnostic, err := run(t, root)
	if err != nil || strings.Contains(diagnostic, "not JUnit XML") {
		t.Fatalf("valid zero-case JUnit: stderr %q, error %v", diagnostic, err)
	}
}

func TestTruthAtHeadAndProseConfirmation(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "checked", "")
	propertyFile(t, root, "prose", "anchors: [code.go]\n")
	write(t, filepath.Join(root, "checked_test.go"), "// spekk: checked\nfunc TestChecked(t *testing.T) {}\n")
	write(t, filepath.Join(root, "code.go"), "package sample\n")
	first := commit(t, root, "Add properties")
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestChecked"/></testsuite>`)
	config(t, root, "cp source.xml result.xml", "result.xml")
	if out, diagnostic, err := run(t, root); err != nil {
		t.Fatal(err, out, diagnostic)
	}
	d := db(t, root)
	if got, _ := truth(t, d, "checked"); got != "verified" {
		t.Fatal(got)
	}
	write(t, filepath.Join(root, "code.go"), "package sample\nvar changed = true\n")
	second := commit(t, root, "Change anchored code")
	if first == second {
		t.Fatal("commit did not advance")
	}
	config(t, root, "true", "absent.xml")
	_, _, _ = run(t, root)
	if got, _ := truth(t, d, "prose"); got != "unconfirmed" {
		t.Fatal(got)
	}
	if got, _ := truth(t, d, "checked"); got != "unchecked" {
		t.Fatal(got)
	}
	var old int
	if err := d.QueryRow(`SELECT COUNT(*) FROM verify_truth t JOIN verify_runs r ON r.id=t.run_id WHERE r."commit"=? AND t.assertion_id='checked' AND t.truth='verified'`, first).Scan(&old); err != nil || old != 1 {
		t.Fatal(old, err)
	}
	propertyFile(t, root, "prose", "anchors: [code.go]\n")
	write(t, filepath.Join(root, "specs", "sample", "assertions", "prose.md"), "---\nid: prose\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\nanchors: [code.go]\n---\n# prose confirmed\n")
	commit(t, root, "Confirm prose")
	_, _, _ = run(t, root)
	if got, _ := truth(t, d, "prose"); got != "unchecked" {
		t.Fatal(got)
	}
}

func TestHistorySurvivesContentRebuildButNotForce(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "")
	write(t, filepath.Join(root, "check_test.go"), "// spekk: check\nfunc TestCheck(t *testing.T) {}\n")
	commit(t, root, "Add check")
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestCheck"/></testsuite>`)
	config(t, root, "cp source.xml result.xml", "result.xml")
	if _, _, err := run(t, root); err != nil {
		t.Fatal(err)
	}
	d := db(t, root)
	write(t, filepath.Join(root, "specs", "sample", "assertions", "check.md"), "---\nid: check\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n---\n# check\n")
	if _, err := index.EnsureFresh(filepath.Join(root, "specs"), index.DBPath(root)); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM verify_runs`).Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if _, err := index.BuildIndex(filepath.Join(root, "specs"), index.DBPath(root), true); err != nil {
		t.Fatal(err)
	}
	if err := d.QueryRow(`SELECT COUNT(*) FROM verify_runs`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if _, _, err := run(t, root); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`PRAGMA user_version = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := index.EnsureFresh(filepath.Join(root, "specs"), index.DBPath(root)); err != nil {
		t.Fatal(err)
	}
	if err := d.QueryRow(`SELECT COUNT(*) FROM verify_runs`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}

func TestLatestRunDirtySkippedAndError(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "")
	write(t, filepath.Join(root, "check_test.go"), "// spekk: check evidence=contract\nfunc TestCheck(t *testing.T) {}\n")
	head := commit(t, root, "Add check")
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestCheck"><skipped/></testcase></testsuite>`)
	config(t, root, "cp source.xml result.xml", "result.xml")
	if _, _, err := run(t, root); err != nil {
		t.Fatal(err)
	}
	d := db(t, root)
	if got, level := truth(t, d, "check"); got != "unchecked" || level != "" {
		t.Fatal(got, level)
	}
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestCheck"><error/></testcase></testsuite>`)
	if _, _, err := run(t, root); err == nil {
		t.Fatal("an error result passed")
	}
	if got, level := truth(t, d, "check"); got != "failing" || level != "" {
		t.Fatal(got, level)
	}
	write(t, filepath.Join(root, "source.xml"), `<testsuite><testcase classname="example.test/acme" name="TestCheck"/></testsuite>`)
	if out, diagnostic, err := run(t, root); err != nil {
		t.Fatal(err, out, diagnostic)
	}
	if got, level := truth(t, d, "check"); got != "verified" || level != "contract" {
		t.Fatal(got, level)
	}
	var count, dirty int
	if err := d.QueryRow(`SELECT COUNT(*), MIN(dirty) FROM verify_runs WHERE "commit"=?`, head).Scan(&count, &dirty); err != nil || count != 3 || dirty != 1 {
		t.Fatal(count, dirty, err)
	}
}

func TestJUnitDoesNotReadExternalEntitiesOrSymlinks(t *testing.T) {
	root := project(t)
	propertyFile(t, root, "check", "")
	commit(t, root, "Add property")
	write(t, filepath.Join(root, "source.xml"), `<!DOCTYPE testsuite [<!ENTITY secret SYSTEM "file:///etc/passwd">]><testsuite><testcase classname="a" name="b">&secret;</testcase></testsuite>`)
	config(t, root, "cp source.xml result.xml", "result.xml")
	_, diagnostic, err := run(t, root)
	if err == nil || !strings.Contains(diagnostic, "directives are not allowed") {
		t.Fatal(diagnostic, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.xml")
	write(t, outside, `<testsuite/>`)
	link := filepath.Join(root, "outside.xml")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	config(t, root, "true", "outside.xml")
	_, _, err = run(t, root)
	if err == nil || !strings.Contains(err.Error(), "outside the project") {
		t.Fatal(err)
	}
}
