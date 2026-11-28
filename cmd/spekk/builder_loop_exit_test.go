package main

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestBuilderLoopExitWhenDone(t *testing.T) {
	bin := buildBuilderLoopCLI(t)

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
			// POSIX sh and a sed that BSD and GNU read the same way: no -i,
			// whose argument differs between them. set -e makes a failing
			// step fail the script, and its stderr reaches the test output.
			fake := "#!/bin/sh\nset -e\nf=specs/sample/assertions/work.md\nsed 's/status: not_started/status: done/' \"$f\" > \"$f.tmp\"\nmv \"$f.tmp\" \"$f\"\n"
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

	t.Run("once waits for interrupted child", func(t *testing.T) {
		dir := builderLoopFixture(t, true)
		fakeDir := t.TempDir()
		fake := "#!/bin/sh\ntrap 'sleep 0.6; printf done > \"$DONE_FILE\"; exit 130' INT\nprintf started > \"$STARTED_FILE\"\nwhile :; do sleep 0.1; done\n"
		if err := os.WriteFile(filepath.Join(fakeDir, "claude"), []byte(fake), 0o755); err != nil {
			t.Fatal(err)
		}
		startedFile := filepath.Join(fakeDir, "started")
		doneFile := filepath.Join(fakeDir, "done")
		outputFile, err := os.Create(filepath.Join(fakeDir, "output"))
		if err != nil {
			t.Fatal(err)
		}
		defer outputFile.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, "builder", "--once")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"), "STARTED_FILE="+startedFile, "DONE_FILE="+doneFile)
		cmd.Stdout = outputFile
		cmd.Stderr = outputFile
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		finished := make(chan error, 1)
		go func() { finished <- cmd.Wait() }()
		tick := time.NewTicker(10 * time.Millisecond)
		defer tick.Stop()
		for {
			if _, err := os.Stat(startedFile); err == nil {
				break
			}
			select {
			case err := <-finished:
				t.Fatalf("parent exited before child started: %v", err)
			case <-ctx.Done():
				t.Fatal("child did not start")
			case <-tick.C:
			}
		}
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-finished:
			if _, statErr := os.Stat(doneFile); statErr != nil {
				t.Fatalf("parent exited before child finished: %v", statErr)
			}
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatalf("interrupted build must exit 1: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("parent did not exit after child")
		}
	})
}

func TestLoopBuilderWaitsForInterruptedChild(t *testing.T) {
	bin := buildBuilderLoopCLI(t)

	dir := builderLoopFixture(t, true)
	fakeDir := t.TempDir()
	fake := `#!/bin/sh
if [ -f "$CHILD_DONE" ]; then
  sed 's/status: not_started/status: done/' "$ASSERTION_FILE" > "$ASSERTION_FILE.tmp" && mv "$ASSERTION_FILE.tmp" "$ASSERTION_FILE"
  printf 'SECOND_BUILD\n'
  exit 0
fi
trap 'sleep 0.3; printf done > "$CHILD_DONE"; printf result > "$RESULT_FILE"; exit 130' INT
printf 'CHILD_READY\n'
i=0
while [ "$i" -lt 40 ]; do
  sleep 0.1
  i=$((i + 1))
done
exit 1
`
	if err := os.WriteFile(filepath.Join(fakeDir, "claude"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	childDone := filepath.Join(fakeDir, "child_done")
	resultFile := filepath.Join(dir, "result.txt")
	assertionFile := filepath.Join(dir, "specs", "sample", "assertions", "work.md")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "loop", "builder")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"), "CHILD_DONE="+childDone, "RESULT_FILE="+resultFile, "ASSERTION_FILE="+assertionFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	interrupted := false
	stopped := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")
		if !interrupted && strings.Contains(line, "CHILD_READY") {
			interrupted = true
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				t.Fatal(err)
			}
		}
		if !stopped && strings.Contains(line, "Waiting for new work") {
			stopped = true
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("loop exited with error: %v\n%s", err, output.String())
	}
	if !interrupted || !stopped || !strings.Contains(output.String(), "SECOND_BUILD") {
		t.Fatalf("loop did not continue after the interrupted child:\n%s", output.String())
	}
	if _, err := os.Stat(childDone); err != nil {
		t.Fatalf("child did not receive SIGINT: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "Builder loop complete. 1 assertion completed.") {
		t.Fatalf("completion count is wrong:\n%s", output.String())
	}
	show := exec.Command("git", "show", "HEAD~1:result.txt")
	show.Dir = dir
	if out, err := show.CombinedOutput(); err != nil || string(out) != "result" {
		t.Fatalf("commit step did not run after the child exited: %v\n%s\n%s", err, out, output.String())
	}
}

// A Ctrl+C that reaches the loop during the commit step must not leave git
// running: the loop waits for the step, then exits. Before the fix the loop
// exited at once, and the orphan git wrote to .git while t.TempDir removed
// it (main CI, 2026-11-27: "unlinkat .../.git: directory not empty").
func TestLoopBuilderInterruptWaitsForCommitStep(t *testing.T) {
	bin := buildBuilderLoopCLI(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := builderLoopFixture(t, true)
	fakeDir := t.TempDir()
	gitDone := filepath.Join(fakeDir, "git_done")
	// The fake builder leaves a change and fails. The git wrapper makes the
	// commit step slow, and records when the last git command of the step
	// exits.
	files := map[string]string{
		"claude": "#!/bin/sh\nprintf partial > partial.txt\nexit 1\n",
		"git": `#!/bin/sh
[ "$1" = status ] && sleep 0.5
"$REAL_GIT" "$@"
rc=$?
[ "$1" = commit ] && printf done > "$GIT_DONE"
exit $rc
`,
	}
	for name, script := range files {
		if err := os.WriteFile(filepath.Join(fakeDir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "loop", "builder")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"), "REAL_GIT="+realGit, "GIT_DONE="+gitDone)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	interrupted := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")
		if !interrupted && strings.Contains(line, "Committing changes...") {
			interrupted = true
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil || !interrupted {
		t.Fatalf("loop must exit 0 after Ctrl+C: %v\n%s", err, output.String())
	}
	if _, err := os.Stat(gitDone); err != nil {
		t.Fatalf("loop exited before the commit step finished: %v\n%s", err, output.String())
	}
	show := exec.Command("git", "show", "HEAD:partial.txt")
	show.Dir = dir
	if out, err := show.CombinedOutput(); err != nil || string(out) != "partial" {
		t.Fatalf("commit step did not complete: %v\n%s\n%s", err, out, output.String())
	}
	if !strings.Contains(output.String(), "Builder loop complete. 0 assertions completed.") {
		t.Fatalf("completion count is wrong:\n%s", output.String())
	}
}

// A signal that reaches the loop while spekk next runs must not let the loop
// start a builder agent: the loop exits after spekk next and leaves no agent
// behind. Before the fix the handler exited while the loop started claude, so
// an orphan builder agent wrote to the repository after the exit.
func TestLoopBuilderSignalDuringNextStartsNoAgent(t *testing.T) {
	bin := buildBuilderLoopCLI(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			// The defective code loses the race in about half of the runs,
			// so several runs make the test fail reliably.
			for run := 0; run < 3; run++ {
				dir := builderLoopFixture(t, true)
				fakeDir := t.TempDir()
				slow := filepath.Join(fakeDir, "slow_started")
				release := filepath.Join(fakeDir, "release")
				agent := filepath.Join(fakeDir, "agent_started")
				// The first git command is the first one of spekk next. The
				// wrapper records its start, then waits until the test lets
				// it continue (5 s at most).
				files := map[string]string{
					"claude": "#!/bin/sh\n: > \"$AGENT_STARTED\"\nexit 1\n",
					"git":    "#!/bin/sh\nif mkdir \"$SLOW_STARTED.lock\" 2>/dev/null; then\n  : > \"$SLOW_STARTED\"\n  i=0\n  while [ ! -e \"$RELEASE\" ] && [ $i -lt 100 ]; do sleep 0.05; i=$((i+1)); done\nfi\nexec \"$REAL_GIT\" \"$@\"\n",
				}
				for name, script := range files {
					if err := os.WriteFile(filepath.Join(fakeDir, name), []byte(script), 0o755); err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, bin, "loop", "builder")
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"), "REAL_GIT="+realGit, "SLOW_STARTED="+slow, "RELEASE="+release, "AGENT_STARTED="+agent)
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				for {
					if _, err := os.Stat(slow); err == nil {
						break
					}
					if ctx.Err() != nil {
						t.Fatal("spekk next did not start git")
					}
					time.Sleep(10 * time.Millisecond)
				}
				if err := cmd.Process.Signal(sig); err != nil {
					t.Fatal(err)
				}
				// Let spekk next continue only after the loop reports that it
				// handles the signal.
				var output strings.Builder
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					line := scanner.Text()
					output.WriteString(line + "\n")
					if strings.Contains(line, "Exiting gracefully") {
						if err := os.WriteFile(release, nil, 0o644); err != nil {
							t.Fatal(err)
						}
					}
				}
				if err := scanner.Err(); err != nil {
					t.Fatal(err)
				}
				err = cmd.Wait()
				// An agent that the loop started just before it exited
				// needs a moment to write its marker.
				time.Sleep(300 * time.Millisecond)
				if _, statErr := os.Stat(agent); statErr == nil {
					t.Fatalf("run %d: builder agent started after %s:\n%s", run, sig, output.String())
				}
				if err != nil || !strings.Contains(output.String(), "Builder loop complete. 0 assertions completed.") {
					t.Fatalf("run %d: loop must exit 0 with the count line: %v\n%s", run, err, output.String())
				}
			}
		})
	}
}

// SIGTERM that reaches only the loop during a build must stop the builder
// agent before the loop exits. Before the fix the loop exited at once with
// status 0, and the agent kept running and writing to the repository. The
// fake agent exits 0, so a count of 1 would show that the loop counted the
// interrupted build.
func TestLoopBuilderSigtermDuringBuildWaitsForAgent(t *testing.T) {
	bin := buildBuilderLoopCLI(t)
	// second is the signal that the test sends while the loop waits for the
	// agent. nil sends none: the agent exits and the loop exits 0.
	for _, second := range []os.Signal{nil, os.Interrupt, syscall.SIGTERM} {
		name := "agent exits"
		if second != nil {
			name = "second " + second.String()
		}
		t.Run(name, func(t *testing.T) {
			dir := builderLoopFixture(t, true)
			fakeDir := t.TempDir()
			pidFile := filepath.Join(fakeDir, "agent_pid")
			termFile := filepath.Join(fakeDir, "agent_term")
			release := filepath.Join(fakeDir, "release")
			doneFile := filepath.Join(fakeDir, "agent_done")
			// The fake agent leaves a change, then waits. On SIGTERM it
			// records the signal and waits until the test releases it (5 s
			// at most), so the loop must wait for it.
			fake := `#!/bin/sh
trap 'printf term > "$AGENT_TERM"; i=0; while [ ! -e "$RELEASE" ] && [ $i -lt 100 ]; do sleep 0.05; i=$((i+1)); done; printf done > "$AGENT_DONE"; exit 0' TERM
printf '%s' "$$" > "$AGENT_PID"
printf partial > partial.txt
printf 'AGENT_READY\n'
i=0
while [ $i -lt 100 ]; do sleep 0.05; i=$((i+1)); done
exit 1
`
			if err := os.WriteFile(filepath.Join(fakeDir, "claude"), []byte(fake), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				// Release an agent that a defective loop left behind, so it
				// does not write to dir while t.TempDir removes it.
				_ = os.WriteFile(release, nil, 0o644)
				if data, err := os.ReadFile(pidFile); err == nil {
					if pid, err := strconv.Atoi(string(data)); err == nil {
						_ = syscall.Kill(pid, syscall.SIGKILL)
					}
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "loop", "builder")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"), "AGENT_PID="+pidFile, "AGENT_TERM="+termFile, "RELEASE="+release, "AGENT_DONE="+doneFile)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			lines := make(chan string)
			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					lines <- scanner.Text()
				}
				close(lines)
			}()
			var output strings.Builder
			// next returns the next output line; ok is false at the end.
			next := func() (string, bool) {
				select {
				case line, ok := <-lines:
					output.WriteString(line + "\n")
					return line, ok
				case <-ctx.Done():
					t.Fatalf("loop timed out:\n%s", output.String())
					return "", false
				}
			}
			for {
				line, ok := next()
				if !ok {
					t.Fatalf("loop exited before the agent started:\n%s", output.String())
				}
				if strings.Contains(line, "AGENT_READY") {
					break
				}
			}
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			for {
				if _, err := os.Stat(termFile); err == nil {
					break
				}
				if ctx.Err() != nil {
					t.Fatalf("builder agent did not receive SIGTERM:\n%s", output.String())
				}
				time.Sleep(10 * time.Millisecond)
			}
			if second != nil {
				if err := cmd.Process.Signal(second); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(release, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if second != nil {
				// The agent still holds the stdout pipe, so the test waits
				// for the loop process and not for the end of its output.
				err = cmd.Wait()
				for range lines {
				}
				out := output.String()
				want := 128 + int(second.(syscall.Signal))
				if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != want {
					t.Fatalf("second %s must exit %d at once: %v\n%s", second, want, err, out)
				}
				if _, statErr := os.Stat(doneFile); statErr == nil {
					t.Fatalf("loop waited for the agent after the second signal:\n%s", out)
				}
				return
			}
			for {
				if _, ok := next(); !ok {
					break
				}
			}
			err = cmd.Wait()
			out := output.String()
			if err != nil {
				t.Fatalf("loop must exit 0 after the agent exits: %v\n%s", err, out)
			}
			if _, err := os.Stat(doneFile); err != nil {
				t.Fatalf("loop exited before the agent exited:\n%s", out)
			}
			data, err := os.ReadFile(pidFile)
			if err != nil {
				t.Fatal(err)
			}
			pid, err := strconv.Atoi(string(data))
			if err != nil {
				t.Fatal(err)
			}
			if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
				t.Fatalf("builder agent %d still runs after the loop exited: %v", pid, err)
			}
			if strings.Contains(out, "Committing changes...") || strings.Count(out, "Launching Builder Agent...") != 1 {
				t.Fatalf("loop must not commit or start another agent after SIGTERM:\n%s", out)
			}
			log := exec.Command("git", "rev-list", "--count", "HEAD")
			log.Dir = dir
			if count, err := log.Output(); err != nil || strings.TrimSpace(string(count)) != "1" {
				t.Fatalf("loop committed the interrupted build: %v %s", err, count)
			}
			if !strings.Contains(out, "Builder loop complete. 0 assertions completed.") {
				t.Fatalf("count must not include the interrupted build:\n%s", out)
			}
		})
	}
}

func buildBuilderLoopCLI(t *testing.T) string {
	t.Helper()
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
	return bin
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
