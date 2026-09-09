package sandbox

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"os/exec"
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
			command := replaceInstallTarget(t, installCommand(user, "printf "+serviceReady), target)
			if output, err := exec.Command("bash", "-c", command).CombinedOutput(); err == nil || strings.Contains(string(output), serviceReady) {
				t.Fatalf("missing upload must stop installation before service commands: %v, output %q", err, output)
			}
			if content, err := os.ReadFile(target); err != nil || !bytes.Equal(content, oldBinary) {
				t.Fatalf("preparation failure changed the installed binary: %v", err)
			}
			probe()
			assertNoTemporaryBinaries(t, target)

			if err := os.WriteFile(filepath.Join(home, stagedBinary), []byte("#!/bin/sh\nprintf replacement\n"), 0o600); err != nil {
				t.Fatal(err)
			}
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

// Keep the generated command intact except for its destination in the test directory.
func replaceInstallTarget(t *testing.T, command, target string) string {
	t.Helper()
	const installedPath = "/opt/spekk/agent-client"
	if strings.Contains(command, "| sudo bash") {
		parts := strings.SplitN(command, "'", 3)
		if len(parts) != 3 {
			t.Fatal("missing encoded installation script")
		}
		script, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			t.Fatal(err)
		}
		parts[1] = base64.StdEncoding.EncodeToString([]byte(strings.ReplaceAll(string(script), installedPath, target)))
		command = strings.Join(parts, "'")
	}
	return strings.ReplaceAll(command, installedPath, target)
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
