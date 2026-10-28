package main

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuilderLoopExitWhenDone(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "spekk")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", bin, "./cmd/spekk")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}

	for _, command := range [][]string{{"builder"}, {"loop", "builder"}} {
		name := strings.Join(command, " ")
		t.Run(name, func(t *testing.T) {
			empty := builderLoopFixture(t, false)
			args := append(append([]string{}, command...), "--exit-when-done")
			out, err := runBuilderLoopCommand(bin, empty, args, 3*time.Second)
			if err != nil || !strings.Contains(out, "Builder loop complete. 0 assertions completed.") {
				t.Fatalf("empty queue: %v\n%s", err, out)
			}

			ready := builderLoopFixture(t, true)
			fakeBin := filepath.Join(t.TempDir(), "claude")
			fake := "#!/bin/sh\nsed -i 's/status: not_started/status: done/' specs/sample/assertions/work.md\n"
			if err := os.WriteFile(fakeBin, []byte(fake), 0o755); err != nil {
				t.Fatal(err)
			}
			out, err = runBuilderLoopCommandWithPath(bin, ready, args, 4*time.Second, filepath.Dir(fakeBin))
			if err != nil || !strings.Contains(out, "Builder loop complete. 1 assertion completed.") {
				t.Fatalf("last assertion: %v\n%s", err, out)
			}

			out, err = interruptWaitingBuilderLoop(bin, empty, command, "Waiting for new work", "")
			if err != nil || !strings.Contains(out, "Waiting for new work") || strings.Count(out, "Builder loop complete. 0 assertions completed.") != 2 {
				t.Fatalf("default loop must wait for work and report count on Ctrl+C: %v\n%s", err, out)
			}

			blocked := builderLoopFixture(t, true)
			assertionFile := filepath.Join(blocked, "specs", "sample", "assertions", "work.md")
			data, err := os.ReadFile(assertionFile)
			if err != nil {
				t.Fatal(err)
			}
			data = []byte(strings.Replace(string(data), "status: not_started", "status: not_started\nbranch: other", 1))
			if err := os.WriteFile(assertionFile, data, 0o644); err != nil {
				t.Fatal(err)
			}
			out, err = interruptWaitingBuilderLoop(bin, blocked, args, "Waiting...", "")
			if err != nil || !strings.Contains(out, "Waiting...") || !strings.Contains(out, "Builder loop complete. 0 assertions completed.") {
				t.Fatalf("ineligible assertion must keep the loop waiting: %v\n%s", err, out)
			}

			failed := builderLoopFixture(t, true)
			failedBin := filepath.Join(t.TempDir(), "claude")
			if err := os.WriteFile(failedBin, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			failureLine := "Build did not succeed."
			if command[0] == "loop" {
				failureLine = "Builder agent exited with error"
			}
			out, err = interruptWaitingBuilderLoop(bin, failed, command, failureLine, filepath.Dir(failedBin))
			if err != nil || !strings.Contains(out, "Builder loop complete. 0 assertions completed.") {
				t.Fatalf("failed build must not increase count: %v\n%s", err, out)
			}
		})
	}
	out, err := runBuilderLoopCommand(bin, builderLoopFixture(t, false), []string{"builder", "--once"}, 3*time.Second)
	if err != nil || !strings.Contains(out, "No assertions to work on.") || strings.Contains(out, "Builder loop complete.") {
		t.Fatalf("--once message changed: %v\n%s", err, out)
	}
}

func interruptWaitingBuilderLoop(bin, dir string, args []string, waitText, path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	if path != "" {
		cmd.Env = append(os.Environ(), "PATH="+path+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	var output strings.Builder
	interrupted := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")
		if !interrupted && strings.Contains(line, waitText) {
			interrupted = true
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				return output.String(), err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return output.String(), err
	}
	return output.String(), cmd.Wait()
}

func builderLoopFixture(t *testing.T, ready bool) string {
	t.Helper()
	dir := t.TempDir()
	assertionDir := filepath.Join(dir, "specs", "sample", "assertions")
	if err := os.MkdirAll(assertionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parent := "---\nid: sample\ncreated: 2026-10-28T00:00:00Z\npriority: 1\n---\n\n# Sample\n"
	assertion := "---\nid: work\nparent: sample\ncreated: 2026-10-28T00:00:00Z\npriority: 1\nstatus: done\n---\n\n# Work\n"
	if ready {
		assertion = strings.Replace(assertion, "status: done", "status: not_started", 1)
	}
	for path, data := range map[string]string{
		filepath.Join(dir, "specs", "sample", "sample.md"): parent,
		filepath.Join(assertionDir, "work.md"):             assertion,
	} {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.email", "test@example.com"}, {"config", "user.name", "Test"}, {"add", "."}, {"commit", "-m", "Initialize"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	return dir
}

func runBuilderLoopCommand(bin, dir string, args []string, limit time.Duration) (string, error) {
	return runBuilderLoopCommandWithPath(bin, dir, args, limit, "")
}

func runBuilderLoopCommandWithPath(bin, dir string, args []string, limit time.Duration, path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	if path != "" {
		cmd.Env = append(os.Environ(), "PATH="+path+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
