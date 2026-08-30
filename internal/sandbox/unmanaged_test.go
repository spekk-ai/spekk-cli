package sandbox

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRegisterMachine(t *testing.T) {
	key := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(key, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("records an absolute key path", func(t *testing.T) {
		// A relative path breaks every later command, which runs from
		// a different working directory than create did.
		dir, file := filepath.Split(key)
		t.Chdir(dir)

		meta := &SandboxMeta{}
		if err := registerMachine(CreateOptions{IP: "1.2.3.4", SSHKey: file}, meta); err != nil {
			t.Fatal(err)
		}
		if !filepath.IsAbs(meta.SSHKeyPath) {
			t.Errorf("SSHKeyPath = %q, want an absolute path", meta.SSHKeyPath)
		}
		if meta.IP != "1.2.3.4" {
			t.Errorf("IP = %q", meta.IP)
		}
	})

	t.Run("refuses before recording anything", func(t *testing.T) {
		for want, opts := range map[string]CreateOptions{
			"--ip":      {SSHKey: key},
			"--ssh-key": {IP: "1.2.3.4"},
			// A key that is not there is a typo, and it should not
			// cost the operator a record they must then force away.
			"no such file": {IP: "1.2.3.4", SSHKey: filepath.Join(t.TempDir(), "absent")},
		} {
			meta := &SandboxMeta{}
			err := registerMachine(opts, meta)
			if err == nil {
				t.Fatalf("expected an error mentioning %q", want)
			}
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q should mention %q", err, want)
			}
			if namesMachine(meta) {
				t.Errorf("nothing should be recorded on failure, got %+v", meta)
			}
		}
	})
}

// Destroying a machine spekk does not own must never delete the operator's
// key, and must not drop the record while the agent may still be running.
func TestDestroyUnmanagedMachine(t *testing.T) {
	operatorKey := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "id_ed25519")
		for _, p := range []string{path, path + ".pub"} {
			if err := os.WriteFile(p, []byte("secret"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return path
	}
	save := func(t *testing.T, key string) {
		t.Helper()
		if err := SaveSandbox("box", &SandboxMeta{Provider: ProviderNone, IP: "1.2.3.4", SSHKeyPath: key}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("keeps the operator's key", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		key := operatorKey(t)
		save(t, key)
		stubStopAgent(t, nil)

		if err := Destroy(nil, "box", true); err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{key, key + ".pub"} {
			if _, err := os.Stat(p); err != nil {
				t.Errorf("destroy deleted %s: %v", p, err)
			}
		}
		if got, _ := GetSandbox("box"); got != nil {
			t.Error("the record should be gone after a clean destroy")
		}
	})

	t.Run("keeps the record when the agent cannot be stopped", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		save(t, operatorKey(t))
		stubStopAgent(t, errors.New("host unreachable"))

		answerPrompt(t, "y\n")
		if err := Destroy(nil, "box", false); err == nil {
			t.Fatal("destroy must fail when the agent cannot be stopped")
		}
		if got, _ := GetSandbox("box"); got == nil {
			t.Error("the record must survive, or an agent keeps running with nothing pointing at it")
		}
	})
}

func stubStopAgent(t *testing.T, err error) {
	t.Helper()
	orig := stopAgent
	stopAgent = func(meta *SandboxMeta, name string) error { return err }
	t.Cleanup(func() { stopAgent = orig })
}

// answerPrompt feeds one answer to the confirmation prompt Destroy reads
// from stdin.
func answerPrompt(t *testing.T, answer string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(answer); err != nil {
		t.Fatal(err)
	}
	w.Close()
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; r.Close() })
}

// The teardown shell decides whether spekk may forget a machine it does not
// own, so its exit status has to mean exactly one thing: every credential is
// off the machine, and the agent is stopped.
//
// The secret paths are pointed at real temp files, so the check that the
// files are gone is exercised against the filesystem rather than simulated.
func TestTeardownCommand(t *testing.T) {
	// run creates the secrets, stubs systemctl and rm on PATH, and runs
	// the real command. rmBody is the body of the stub rm.
	run := func(t *testing.T, systemctl, rmBody string) (leftBehind []string, ok bool) {
		t.Helper()
		dir := t.TempDir()

		var secrets []string
		for _, name := range []string{"agent.env", "git-credentials", "gh"} {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
				t.Fatal(err)
			}
			secrets = append(secrets, path)
		}
		orig := agentSecrets
		agentSecrets = secrets
		t.Cleanup(func() { agentSecrets = orig })

		bin := t.TempDir()
		for name, body := range map[string]string{"systemctl": systemctl, "rm": rmBody} {
			if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/bash\n"+body+"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command("bash", "-c", teardownCommand())
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
		err := cmd.Run()

		for _, path := range secrets {
			if _, statErr := os.Stat(path); statErr == nil {
				leftBehind = append(leftBehind, filepath.Base(path))
			}
		}
		return leftBehind, err == nil
	}

	// A unit that was never installed is the state any create that died
	// before deployAgent leaves behind. The credentials are already on the
	// machine by then, so they still have to come off.
	unitAbsent := `[ "$1" = "is-active" ] && exit 3; exit 5`
	unitStopped := `[ "$1" = "is-active" ] && exit 3; exit 0`
	agentRunning := `exit 0`

	realRm := `command /bin/rm "$@"`
	// An rm that reports success and removes nothing. Trusting its exit
	// status would report a clean teardown over live credentials.
	lyingRm := `exit 0`
	failingRm := `exit 1`

	t.Run("removes every secret when the unit was never installed", func(t *testing.T) {
		leftBehind, ok := run(t, unitAbsent, realRm)
		if !ok {
			t.Error("teardown must succeed when the unit was never installed")
		}
		if len(leftBehind) > 0 {
			t.Errorf("left on the machine: %v", leftBehind)
		}
	})

	// Reporting success for any of these would let Destroy delete the
	// local record while a credential stays on a machine spekk no longer
	// knows about.
	t.Run("fails when rm reports success but removes nothing", func(t *testing.T) {
		if _, ok := run(t, unitStopped, lyingRm); ok {
			t.Error("teardown trusted rm instead of checking the files were gone")
		}
	})

	t.Run("fails when rm cannot remove", func(t *testing.T) {
		if _, ok := run(t, unitStopped, failingRm); ok {
			t.Error("teardown reported success although a credential is still on the machine")
		}
	})

	t.Run("fails when rm removes the file but reports failure", func(t *testing.T) {
		// The outcome check cannot see this: a path whose parent is
		// unreadable looks absent. rm's own status is the only signal
		// left, so both guards are needed.
		if _, ok := run(t, unitStopped, `command /bin/rm "$@"; exit 1`); ok {
			t.Error("teardown ignored rm's failure")
		}
	})

	t.Run("fails while the agent is still running", func(t *testing.T) {
		if _, ok := run(t, agentRunning, realRm); ok {
			t.Error("teardown reported success although the agent is still active")
		}
	})
}

// agentSecrets is an inventory of every place spekk writes a credential on a
// sandbox. It is asserted explicitly because the cost of quietly losing an
// entry is a live token left on a machine spekk has forgotten, and because
// nothing else in the tests would notice a shorter list.
func TestAgentSecretsInventory(t *testing.T) {
	want := []string{
		// injectCredentials writes AWS keys, GITHUB_TOKEN, agent token.
		"/etc/spekk/agent.env",
		// configureGitCredentials writes the token twice: once here...
		"/home/agent/.git-credentials",
		// ...and once here, through gh auth login --with-token.
		"/home/agent/.config/gh",
	}
	if !slices.Equal(agentSecrets, want) {
		t.Errorf("agentSecrets = %v, want %v", agentSecrets, want)
	}
}
