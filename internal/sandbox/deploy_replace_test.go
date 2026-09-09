package sandbox

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestInstallCommandReplacesRunningExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the remote installation script requires a Unix shell")
	}
	const testAgentSeconds = "60"
	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(t.TempDir(), "agent-client")
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Fatal(err)
	}
	oldBinary, err := os.ReadFile(sleep)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, oldBinary, 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	running := exec.Command(target, testAgentSeconds)
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = running.Process.Kill()
		_ = running.Wait()
	})
	command := strings.ReplaceAll(installCommand("root", "printf service-ready"), "/opt/spekk/agent-client", target)
	if output, err := exec.Command("bash", "-c", command).CombinedOutput(); err == nil || strings.Contains(string(output), "service-ready") {
		t.Fatalf("missing upload must stop installation before service commands: %v, output %q", err, output)
	}
	if content, err := os.ReadFile(target); err != nil || !bytes.Equal(content, oldBinary) {
		t.Fatalf("preparation failure changed the installed binary: %v", err)
	}
	assertNoTemporaryBinaries(t, target)

	if err := os.WriteFile(filepath.Join(home, stagedBinary), []byte("#!/bin/sh\nprintf replacement\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("bash", "-c", command).CombinedOutput(); err != nil || string(output) != "service-ready" {
		t.Fatalf("installation failed: %v, output %q", err, output)
	}
	after, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("installation must replace the executable inode")
	}
	if err := running.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("replacement stopped the running executable: %v", err)
	}
	if output, err := exec.Command(target).CombinedOutput(); err != nil || string(output) != "replacement" {
		t.Fatalf("the next launch must use the replacement: %v, output %q", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, stagedBinary)); !os.IsNotExist(err) {
		t.Fatalf("successful installation must remove the uploaded file: %v", err)
	}
	assertNoTemporaryBinaries(t, target)
}

func assertNoTemporaryBinaries(t *testing.T, target string) {
	t.Helper()
	paths, err := filepath.Glob(target + ".*")
	if err != nil || len(paths) != 0 {
		t.Fatalf("temporary binaries remain: %v (%v)", paths, err)
	}
}
