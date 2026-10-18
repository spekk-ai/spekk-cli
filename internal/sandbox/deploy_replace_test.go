package sandbox

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const installProbeEnv = "SPEKK_TEST_INSTALL_PROBE"

func TestInstallProbeProcess(t *testing.T) {
	if os.Getenv(installProbeEnv) != "1" {
		return
	}
	if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
		t.Fatal(err)
	}
}

func TestInstallCommandReplacesRunningExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the remote installation script requires a Unix shell")
	}
	const serviceReady = "service-ready"
	sudoDir := t.TempDir()
	// This models sudoers rules that replace both the working directory and HOME.
	if err := os.WriteFile(filepath.Join(sudoDir, "sudo"), []byte("#!/bin/sh\ncd / || exit 1\nHOME=/\nexport HOME\nexec \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", sudoDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	oldBinary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}

	for _, user := range []string{"root", "ubuntu"} {
		t.Run(user, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "login user's home")
			if err := os.Mkdir(home, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", home)
			target := filepath.Join(t.TempDir(), "agent-client")
			if err := os.WriteFile(target, oldBinary, 0o755); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(target)
			if err != nil {
				t.Fatal(err)
			}
			probe := startInstallProbe(t, target)
			probe()
			command := localInstallCommand(t, installCommand(user, sha256Hex(replacementAgent), sha256Hex(replacementCLI), "printf "+serviceReady), target, home)
			if output, err := exec.Command("bash", "-c", command).CombinedOutput(); err == nil || strings.Contains(string(output), serviceReady) {
				t.Fatalf("missing upload must stop installation before service commands: %v, output %q", err, output)
			}
			if content, err := os.ReadFile(target); err != nil || !bytes.Equal(content, oldBinary) {
				t.Fatalf("preparation failure changed the installed binary: %v", err)
			}
			probe()
			assertNoTemporaryBinaries(t, target)

			stageUploads(t, home)
			if output, err := exec.Command("bash", "-c", command).CombinedOutput(); err != nil || string(output) != serviceReady {
				t.Fatalf("installation failed: %v, output %q", err, output)
			}
			after, err := os.Stat(target)
			if err != nil {
				t.Fatal(err)
			}
			if os.SameFile(before, after) {
				t.Fatal("installation must replace the executable inode")
			}
			probe()
			if output, err := exec.Command(target).CombinedOutput(); err != nil || string(output) != "replacement" {
				t.Fatalf("the next launch must use the replacement: %v, output %q", err, output)
			}
			if _, err := os.Stat(filepath.Join(home, stagedBinary)); !os.IsNotExist(err) {
				t.Fatalf("successful installation must remove the uploaded file: %v", err)
			}
			assertNoTemporaryBinaries(t, target)
		})
	}
}

// The uploads of the tests, and the sha256 that the release publishes for them.
const (
	replacementAgent = "#!/bin/sh\nprintf replacement\n"
	replacementCLI   = "#!/bin/sh\necho v9.9.9\n"
)

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// stageUploads writes both uploads to home, where scp puts them.
func stageUploads(t *testing.T, home string) {
	t.Helper()
	for name, content := range map[string]string{stagedBinary: replacementAgent, stagedCLI: replacementCLI} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// localInstallCommand keeps the generated command intact except for the
// machine paths and the owner: the agent goes to target, the CLI to cli/spekk
// beside it, the PATH link to bin-spekk beside it, and the CLI belongs to the
// user that runs the test, whose home is ownerHome.
func localInstallCommand(t *testing.T, command, target, ownerHome string) string {
	t.Helper()
	if _, err := exec.LookPath("sha256sum"); err != nil {
		t.Skip("the remote installation script needs sha256sum")
	}
	me, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(target)
	replacer := strings.NewReplacer(
		`installed="/opt/spekk/agent-client"`, `installed="`+target+`"`,
		`cli="/opt/spekk/cli/spekk"`, `cli="`+filepath.Join(dir, "cli", "spekk")+`"`,
		`cli_link="/usr/local/bin/spekk"`, `cli_link="`+filepath.Join(dir, "bin-spekk")+`"`,
		`owner="agent"`, `owner="`+me.Username+`"`,
		`owner_home=$(getent passwd "$owner" | cut -d: -f6)`, `owner_home="`+ownerHome+`"`,
	)
	if strings.Contains(command, "| sudo bash") {
		parts := strings.SplitN(command, "'", 3)
		if len(parts) != 3 {
			t.Fatal("missing encoded installation script")
		}
		script, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			t.Fatal(err)
		}
		parts[1] = base64.StdEncoding.EncodeToString([]byte(replacer.Replace(string(script))))
		return strings.Join(parts, "'")
	}
	return replacer.Replace(command)
}

func startInstallProbe(t *testing.T, target string) func() {
	t.Helper()
	running := exec.Command(target, "-test.run=^TestInstallProbeProcess$")
	running.Env = append(os.Environ(), installProbeEnv+"=1")
	input, err := running.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = input.Close() })
	output, childOutput, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = output.Close(); _ = childOutput.Close() })
	running.Stdout = childOutput
	running.Stderr = os.Stderr
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = running.Process.Kill(); _ = running.Wait() })
	if err := childOutput.Close(); err != nil {
		t.Fatal(err)
	}
	return func() {
		t.Helper()
		const probeMessage = "alive\n"
		const probeTimeout = 5 * time.Second
		if err := output.SetReadDeadline(time.Now().Add(probeTimeout)); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(input, probeMessage); err != nil {
			t.Fatalf("agent probe failed: %v", err)
		}
		response := make([]byte, len(probeMessage))
		if _, err := io.ReadFull(output, response); err != nil || string(response) != probeMessage {
			t.Fatalf("agent did not answer the probe: %q (%v)", response, err)
		}
	}
}

func assertNoTemporaryBinaries(t *testing.T, target string) {
	t.Helper()
	paths, err := filepath.Glob(target + ".*")
	if err != nil || len(paths) != 0 {
		t.Fatalf("temporary binaries remain: %v (%v)", paths, err)
	}
}

// A machine an operator provisioned by hand may not carry the install
// directories. The script creates them before it writes the temporary files.
// With the two lines in the other order `set -e` aborts at mktemp, and the
// mkdir downstream of the failure never runs, so no retry heals the machine.
// The CLI lands beside the agent, linked from the PATH, and an old copy in
// the owner's ~/.local/bin becomes a link to it.
func TestInstallCommandInstallsBothIntoMissingDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the remote installation script requires a Unix shell")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	stageUploads(t, home)
	oldCopy := filepath.Join(home, ".local", "bin", "spekk")
	if err := os.MkdirAll(filepath.Dir(oldCopy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldCopy, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A space in the path also proves the script quotes the install path.
	target := filepath.Join(t.TempDir(), "absent dir", "agent-client")
	command := localInstallCommand(t, installCommand("root", sha256Hex(replacementAgent), sha256Hex(replacementCLI), "printf ready"), target, home)
	if output, err := exec.Command("bash", "-c", command).CombinedOutput(); err != nil {
		t.Fatalf("installation into a missing directory failed: %v, output %q", err, output)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the installed executable is missing: %v", err)
	}
	cli := filepath.Join(filepath.Dir(target), "cli", "spekk")
	for _, link := range []string{filepath.Join(filepath.Dir(target), "bin-spekk"), oldCopy} {
		if out, err := exec.Command(link).Output(); err != nil || string(out) != "v9.9.9\n" {
			t.Errorf("%s must run the installed CLI %s: %v, output %q", link, cli, err, out)
		}
	}
}

// A file that does not match the release sha256 replaces nothing: not the
// agent, not the CLI, and the service setup does not run.
func TestInstallCommandRefusesAWrongSHA256(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the remote installation script requires a Unix shell")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	stageUploads(t, home)
	target := filepath.Join(t.TempDir(), "agent-client")
	if err := os.WriteFile(target, []byte("old agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := localInstallCommand(t, installCommand("root", sha256Hex(replacementAgent), sha256Hex("another CLI"), "printf ready"), target, home)
	output, err := exec.Command("bash", "-c", command).CombinedOutput()
	if err == nil || strings.Contains(string(output), "ready") {
		t.Fatalf("a wrong sha256 must stop the installation: %v, output %q", err, output)
	}
	if want := "spekk: " + stagedCLI + " does not match the release sha256"; !strings.Contains(string(output), want) {
		t.Errorf("output %q must contain %q", output, want)
	}
	if content, _ := os.ReadFile(target); string(content) != "old agent" {
		t.Errorf("the agent was replaced: %q", content)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(target), "cli")); !os.IsNotExist(err) {
		t.Errorf("the CLI directory was made: %v", err)
	}
	for _, name := range []string{stagedBinary, stagedCLI} {
		if _, err := os.Stat(filepath.Join(home, name)); !os.IsNotExist(err) {
			t.Errorf("the upload %s must be removed: %v", name, err)
		}
	}
}
