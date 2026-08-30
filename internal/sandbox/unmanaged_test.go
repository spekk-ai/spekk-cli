package sandbox

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

// The teardown shell must take the credentials off the machine even when the
// agent service was never installed — which is the state any create that
// failed before deployAgent leaves behind.
func TestTeardownRemovesSecretsWithoutTheService(t *testing.T) {
	dir := t.TempDir()
	// A systemctl that reports the unit does not exist, and an rm that
	// records what it was asked to remove instead of removing it.
	stub := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/bash\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub("systemctl", `[ "$1" = "cat" ] && exit 1; [ "$1" = "is-active" ] && exit 3; echo "systemctl $* should not run" >&2; exit 5`)
	stub("rm", `for a in "$@"; do case "$a" in -*) ;; *) echo "$a" >> "`+dir+`/removed" ;; esac; done`)

	cmd := exec.Command("bash", "-c", teardownCommand())
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("teardown must succeed when the unit was never installed: %v\n%s", err, out)
	}

	removed, err := os.ReadFile(filepath.Join(dir, "removed"))
	if err != nil {
		t.Fatal("teardown removed nothing")
	}
	for _, path := range []string{"/etc/spekk/agent.env", "/home/agent/.git-credentials", "/home/agent/.config/gh"} {
		if !strings.Contains(string(removed), path) {
			t.Errorf("teardown left %s on the machine", path)
		}
	}
}
